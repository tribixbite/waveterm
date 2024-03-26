package screen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/dbutil"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbus"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
)

func GetScreenById(ctx context.Context, screenId string) (*ScreenType, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (*ScreenType, error) {
		query := `SELECT * FROM screen WHERE screenid = ?`
		screen := dbutil.GetMapGen[*ScreenType](tx, query, screenId)
		return screen, nil
	})
}

func GetScreenLinesById(ctx context.Context, screenId string) (*ScreenLinesType, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (*ScreenLinesType, error) {
		query := `SELECT screenid FROM screen WHERE screenid = ?`
		screen := dbutil.GetMappable[*ScreenLinesType](tx, query, screenId)
		if screen == nil {
			return nil, nil
		}
		query = `SELECT * FROM line WHERE screenid = ? ORDER BY linenum`
		screen.Lines = dbutil.SelectMappable[*sstore.LineType](tx, query, screen.ScreenId)
		query = `SELECT * FROM cmd WHERE screenid = ?`
		screen.Cmds = dbutil.SelectMapsGen[*sstore.CmdType](tx, query, screen.ScreenId)
		return screen, nil
	})
}

// includes archived screens
func GetSessionScreens(ctx context.Context, sessionId string) ([]*ScreenType, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) ([]*ScreenType, error) {
		query := `SELECT * FROM screen WHERE sessionid = ? ORDER BY archived, screenidx, archivedts`
		rtn := dbutil.SelectMapsGen[*ScreenType](tx, query, sessionId)
		return rtn, nil
	})
}

// screen may not exist at this point (so don't query screen table)
func cleanScreenCmds(ctx context.Context, screenId string) error {
	var removedCmds []string
	txErr := WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT lineid FROM cmd WHERE screenid = ? AND lineid NOT IN (SELECT lineid FROM line WHERE screenid = ?)`
		removedCmds = tx.SelectStrings(query, screenId, screenId)
		query = `DELETE FROM cmd WHERE screenid = ? AND lineid NOT IN (SELECT lineid FROM line WHERE screenid = ?)`
		tx.Exec(query, screenId, screenId)
		return nil
	})
	if txErr != nil {
		return txErr
	}
	for _, lineId := range removedCmds {
		DeletePtyOutFile(ctx, screenId, lineId)
	}
	return nil
}

func ArchiveScreen(ctx context.Context, sessionId string, screenId string) (scbus.UpdatePacket, error) {
	var isActive bool
	txErr := WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT screenid FROM screen WHERE sessionid = ? AND screenid = ?`
		if !tx.Exists(query, sessionId, screenId) {
			return fmt.Errorf("cannot close screen (not found)")
		}
		if isWebShare(tx, screenId) {
			return fmt.Errorf("cannot archive screen while web-sharing.  stop web-sharing before trying to archive.")
		}
		query = `SELECT archived FROM screen WHERE sessionid = ? AND screenid = ?`
		closeVal := tx.GetBool(query, sessionId, screenId)
		if closeVal {
			return nil
		}
		query = `SELECT count(*) FROM screen WHERE sessionid = ? AND NOT archived`
		numScreens := tx.GetInt(query, sessionId)
		if numScreens <= 1 {
			return fmt.Errorf("cannot archive the last screen in a session")
		}
		query = `UPDATE screen SET archived = 1, archivedts = ?, screenidx = 0 WHERE sessionid = ? AND screenid = ?`
		tx.Exec(query, time.Now().UnixMilli(), sessionId, screenId)
		isActive = tx.Exists(`SELECT sessionid FROM session WHERE sessionid = ? AND activescreenid = ?`, sessionId, screenId)
		if isActive {
			screenIds := tx.SelectStrings(`SELECT screenid FROM screen WHERE sessionid = ? AND NOT archived ORDER BY screenidx`, sessionId)
			nextId := getNextId(screenIds, screenId)
			tx.Exec(`UPDATE session SET activescreenid = ? WHERE sessionid = ?`, nextId, sessionId)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	newScreen, err := GetScreenById(ctx, screenId)
	if err != nil {
		return nil, fmt.Errorf("cannot retrive archived screen: %w", err)
	}
	update := scbus.MakeUpdatePacket()
	update.AddUpdate(*newScreen)
	if isActive {
		bareSession, err := GetBareSessionById(ctx, sessionId)
		if err != nil {
			return nil, err
		}
		update.AddUpdate(*bareSession)
	}
	return update, nil
}

func UnArchiveScreen(ctx context.Context, sessionId string, screenId string) error {
	txErr := WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT screenid FROM screen WHERE sessionid = ? AND screenid = ? AND archived`
		if !tx.Exists(query, sessionId, screenId) {
			return fmt.Errorf("cannot re-open screen (not found or not archived)")
		}
		maxScreenIdx := tx.GetInt(`SELECT COALESCE(max(screenidx), 0) FROM screen WHERE sessionid = ? AND NOT archived`, sessionId)
		query = `UPDATE screen SET archived = 0, screenidx = ? WHERE sessionid = ? AND screenid = ?`
		tx.Exec(query, maxScreenIdx+1, sessionId, screenId)
		return nil
	})
	return txErr
}

func insertScreenUpdate(tx *sstore.TxWrap, screenId string, updateType string) {
	if screenId == "" {
		tx.SetErr(errors.New("invalid screen-update, screenid is empty"))
		return
	}
	nowTs := time.Now().UnixMilli()
	query := `INSERT INTO screenupdate (screenid, lineid, updatetype, updatets) VALUES (?, ?, ?, ?)`
	tx.Exec(query, screenId, "", updateType, nowTs)
	sstore.NotifyUpdateWriter()
}

func insertScreenNewUpdate(tx *sstore.TxWrap, screenId string) {
	nowTs := time.Now().UnixMilli()
	query := `INSERT INTO screenupdate (screenid, lineid, updatetype, updatets)
              SELECT screenid, lineid, ?, ? FROM line WHERE screenid = ? AND NOT archived ORDER BY linenum DESC`
	tx.Exec(query, sstore.UpdateType_LineNew, nowTs, screenId)
	query = `INSERT INTO screenupdate (screenid, lineid, updatetype, updatets)
             SELECT c.screenid, c.lineid, ?, ? FROM cmd c, line l WHERE c.screenid = ? AND l.lineid = c.lineid AND NOT l.archived ORDER BY l.linenum DESC`
	tx.Exec(query, sstore.UpdateType_PtyPos, nowTs, screenId)
	sstore.NotifyUpdateWriter()
}

func handleScreenDelUpdate(tx *sstore.TxWrap, screenId string) {
	query := `DELETE FROM screenupdate WHERE screenid = ?`
	tx.Exec(query, screenId)
	query = `DELETE FROM webptypos WHERE screenid = ?`
	tx.Exec(query, screenId)
	// don't insert UpdateType_ScreenDel (we already processed it in cmdrunner)
}

func insertScreenDelUpdate(tx *sstore.TxWrap, screenId string) {
	handleScreenDelUpdate(tx, screenId)
	insertScreenUpdate(tx, screenId, sstore.UpdateType_ScreenDel)
	// don't insert UpdateType_ScreenDel (we already processed it in cmdrunner)
}

func GetScreenUpdates(ctx context.Context, maxNum int) ([]*ScreenUpdateType, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) ([]*ScreenUpdateType, error) {
		var updates []*ScreenUpdateType
		query := `SELECT * FROM screenupdate ORDER BY updateid LIMIT ?`
		tx.Select(&updates, query, maxNum)
		return updates, nil
	})
}

func RemoveScreenUpdate(ctx context.Context, updateId int64) error {
	if updateId < 0 {
		return nil // in-memory updates (not from DB)
	}
	return sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `DELETE FROM screenupdate WHERE updateid = ?`
		tx.Exec(query, updateId)
		return nil
	})
}

func CountScreenUpdates(ctx context.Context) (int, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (int, error) {
		query := `SELECT count(*) FROM screenupdate`
		return tx.GetInt(query), nil
	})
}

func RemoveScreenUpdates(ctx context.Context, updateIds []int64) error {
	return sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `DELETE FROM screenupdate WHERE updateid IN (SELECT value FROM json_each(?))`
		tx.Exec(query, dbutil.QuickJsonArr(updateIds))
		return nil
	})
}
