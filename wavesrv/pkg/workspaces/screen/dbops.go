package screen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/dbutil"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
)

func GetScreenById(ctx context.Context, screenId string) (*ScreenType, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (*ScreenType, error) {
		query := `SELECT * FROM screen WHERE screenid = ?`
		screen := dbutil.GetMapGen[*ScreenType](tx, query, screenId)
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
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
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
		sstore.DeletePtyOutFile(ctx, screenId, lineId)
	}
	return nil
}

func UnArchiveScreen(ctx context.Context, sessionId string, screenId string) error {
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
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

func InsertScreenDelUpdate(tx *sstore.TxWrap, screenId string) {
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
