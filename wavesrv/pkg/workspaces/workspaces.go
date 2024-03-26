package workspaces

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/dbutil"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbase"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbus"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/workspaces/screen"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/workspaces/session"
)

type ConnectUpdate struct {
	Sessions                 []*session.SessionType                 `json:"sessions,omitempty"`
	Screens                  []*screen.ScreenType                   `json:"screens,omitempty"`
	Remotes                  []*sstore.RemoteRuntimeState           `json:"remotes,omitempty"`
	ScreenStatusIndicators   []*sstore.ScreenStatusIndicatorType    `json:"screenstatusindicators,omitempty"`
	ScreenNumRunningCommands []*sstore.ScreenNumRunningCommandsType `json:"screennumrunningcommands,omitempty"`
	ActiveSessionId          string                                 `json:"activesessionid,omitempty"`
}

func (ConnectUpdate) GetType() string {
	return "connect"
}

// Get all sessions and screens, including remotes
func GetConnectUpdate(ctx context.Context) (*ConnectUpdate, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (*ConnectUpdate, error) {
		update := &ConnectUpdate{}
		sessions := []*session.SessionType{}
		tx.Select(&sessions, session.GetAllSessionsQuery)
		sessionMap := make(map[string]*session.SessionType)
		for _, session := range sessions {
			sessionMap[session.SessionId] = session
			update.Sessions = append(update.Sessions, session)
		}
		query := `SELECT * FROM screen ORDER BY archived, screenidx, archivedts`
		screens := dbutil.SelectMapsGen[*screen.ScreenType](tx, query)
		for _, screen := range screens {
			update.Screens = append(update.Screens, screen)
		}
		query = `SELECT * FROM remote_instance`
		riArr := dbutil.SelectMapsGen[*sstore.RemoteInstance](tx, query)
		for _, ri := range riArr {
			s := sessionMap[ri.SessionId]
			if s != nil {
				s.Remotes = append(s.Remotes, ri)
			}
		}
		query = `SELECT activesessionid FROM client`
		update.ActiveSessionId = tx.GetString(query)
		return update, nil
	})
}

func InsertScreen(ctx context.Context, sessionId string, origScreenName string, opts screen.ScreenCreateOpts, activate bool) (*scbus.ModelUpdatePacketType, error) {
	var newScreenId string
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT sessionid FROM session WHERE sessionid = ? AND NOT archived`
		if !tx.Exists(query, sessionId) {
			return fmt.Errorf("cannot create screen, no session found (or session archived)")
		}
		localRemoteId := tx.GetString(`SELECT remoteid FROM remote WHERE remotealias = ?`, sstore.LocalRemoteAlias)
		if localRemoteId == "" {
			return fmt.Errorf("cannot create screen, no local remote found")
		}
		maxScreenIdx := tx.GetInt(`SELECT COALESCE(max(screenidx), 0) FROM screen WHERE sessionid = ? AND NOT archived`, sessionId)
		var screenName string
		if origScreenName == "" {
			screenNames := tx.SelectStrings(`SELECT name FROM screen WHERE sessionid = ? AND NOT archived`, sessionId)
			screenName = sstore.FmtUniqueName("", "s%d", maxScreenIdx+1, screenNames)
		} else {
			screenName = origScreenName
		}
		var baseScreen *screen.ScreenType
		if opts.HasCopy() {
			if opts.BaseScreenId == "" {
				return fmt.Errorf("invalid screen create opts, copy option with no base screen specified")
			}
			var err error
			baseScreen, err = screen.GetScreenById(tx.Context(), opts.BaseScreenId)
			if err != nil {
				return err
			}
			if baseScreen == nil {
				return fmt.Errorf("cannot create screen, base screen not found")
			}
		}
		newScreenId = scbase.GenWaveUUID()
		screen := &screen.ScreenType{
			SessionId:    sessionId,
			ScreenId:     newScreenId,
			Name:         screenName,
			ScreenIdx:    int64(maxScreenIdx) + 1,
			ScreenOpts:   screen.ScreenOptsType{},
			OwnerId:      "",
			ShareMode:    sstore.ShareModeLocal,
			CurRemote:    sstore.RemotePtrType{RemoteId: localRemoteId},
			NextLineNum:  1,
			SelectedLine: 0,
			Anchor:       screen.ScreenAnchorType{},
			FocusType:    screen.ScreenFocusInput,
			Archived:     false,
			ArchivedTs:   0,
		}
		query = `INSERT INTO screen ( sessionid, screenid, name, screenidx, screenopts, screenviewopts, ownerid, sharemode, webshareopts, curremoteownerid, curremoteid, curremotename, nextlinenum, selectedline, anchor, focustype, archived, archivedts)
                             VALUES (:sessionid,:screenid,:name,:screenidx,:screenopts,:screenviewopts,:ownerid,:sharemode,:webshareopts,:curremoteownerid,:curremoteid,:curremotename,:nextlinenum,:selectedline,:anchor,:focustype,:archived,:archivedts)`
		tx.NamedExec(query, screen.ToMap())
		if activate {
			query = `UPDATE session SET activescreenid = ? WHERE sessionid = ?`
			tx.Exec(query, newScreenId, sessionId)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	newScreen, err := screen.GetScreenById(ctx, newScreenId)
	if err != nil {
		return nil, err
	}
	update := scbus.MakeUpdatePacket()
	update.AddUpdate(*newScreen)
	if activate {
		bareSession, err := session.GetBareSessionById(ctx, sessionId)
		if err != nil {
			return nil, txErr
		}
		update.AddUpdate(*bareSession)
		sstore.UpdateWithCurrentOpenAICmdInfoChat(newScreenId, update)
	}
	return update, nil
}

// if sessionDel is passed, we do *not* delete the screen directory (session delete will handle that)
func DeleteScreen(ctx context.Context, screenId string, sessionDel bool, update *scbus.ModelUpdatePacketType) (*scbus.ModelUpdatePacketType, error) {
	var sessionId string
	var isActive bool
	var screenTombstone *screen.ScreenTombstoneType
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		screen, err := screen.GetScreenById(tx.Context(), screenId)
		if err != nil {
			return fmt.Errorf("cannot get screen to delete: %w", err)
		}
		if screen == nil {
			return fmt.Errorf("cannot delete screen (not found)")
		}
		webSharing := sstore.IsWebShare(tx, screenId)
		if !sessionDel {
			query := `SELECT sessionid FROM screen WHERE screenid = ?`
			sessionId = tx.GetString(query, screenId)
			if sessionId == "" {
				return fmt.Errorf("cannot delete screen (no sessionid)")
			}
			query = `SELECT count(*) FROM screen WHERE sessionid = ? AND NOT archived`
			numScreens := tx.GetInt(query, sessionId)
			if numScreens <= 1 {
				return fmt.Errorf("cannot delete the last screen in a session")
			}
			isActive = tx.Exists(`SELECT sessionid FROM session WHERE sessionid = ? AND activescreenid = ?`, sessionId, screenId)
			if isActive {
				screenIds := tx.SelectStrings(`SELECT screenid FROM screen WHERE sessionid = ? AND NOT archived ORDER BY screenidx`, sessionId)
				nextId := getNextId(screenIds, screenId)
				tx.Exec(`UPDATE session SET activescreenid = ? WHERE sessionid = ?`, nextId, sessionId)
			}
		}
		screenTombstone = &ScreenTombstoneType{
			ScreenId:   screen.ScreenId,
			SessionId:  screen.SessionId,
			Name:       screen.Name,
			DeletedTs:  time.Now().UnixMilli(),
			ScreenOpts: screen.ScreenOpts,
		}
		query := `INSERT INTO screen_tombstone ( screenid, sessionid, name, deletedts, screenopts)
		                                VALUES (:screenid,:sessionid,:name,:deletedts,:screenopts)`
		tx.NamedExec(query, dbutil.ToDBMap(screenTombstone, false))
		query = `DELETE FROM screen WHERE screenid = ?`
		tx.Exec(query, screenId)
		query = `DELETE FROM line WHERE screenid = ?`
		tx.Exec(query, screenId)
		query = `DELETE FROM cmd WHERE screenid = ?`
		tx.Exec(query, screenId)
		query = `UPDATE history SET lineid = '', linenum = 0 WHERE screenid = ?`
		tx.Exec(query, screenId)
		if webSharing {
			insertScreenDelUpdate(tx, screenId)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	if !sessionDel {
		GoDeleteScreenDirs(screenId)
	}
	if update == nil {
		update = scbus.MakeUpdatePacket()
	}
	update.AddUpdate(*screenTombstone)
	update.AddUpdate(ScreenType{SessionId: sessionId, ScreenId: screenId, Remove: true})
	if isActive {
		bareSession, err := GetBareSessionById(ctx, sessionId)
		if err != nil {
			return nil, err
		}
		update.AddUpdate(*bareSession)
	}
	return update, nil
}

// returns sessionId
// if sessionName == "", it will be generated
func InsertSessionWithName(ctx context.Context, sessionName string, activate bool) (*scbus.ModelUpdatePacketType, error) {
	var newScreen *screen.ScreenType
	newSessionId := scbase.GenWaveUUID()
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		names := tx.SelectStrings(`SELECT name FROM session`)
		sessionName = sstore.FmtUniqueName(sessionName, "workspace-%d", len(names)+1, names)
		maxSessionIdx := tx.GetInt(`SELECT COALESCE(max(sessionidx), 0) FROM session`)
		query := `INSERT INTO session (sessionid, name, activescreenid, sessionidx, notifynum, archived, archivedts, sharemode)
                               VALUES (?,         ?,    '',             ?,          0,         0,        0,          ?)`
		tx.Exec(query, newSessionId, sessionName, maxSessionIdx+1, sstore.ShareModeLocal)
		screenUpdate, err := InsertScreen(tx.Context(), newSessionId, "", screen.ScreenCreateOpts{}, true)
		if err != nil {
			return err
		}
		screenUpdateItems := scbus.GetUpdateItems[screen.ScreenType](screenUpdate)
		if len(screenUpdateItems) < 1 {
			return fmt.Errorf("no screen update items")
		}
		newScreen = screenUpdateItems[0]
		if activate {
			query = `UPDATE client SET activesessionid = ?`
			tx.Exec(query, newSessionId)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	sess, err := session.GetSessionById(ctx, newSessionId)
	if err != nil {
		return nil, err
	}
	update := scbus.MakeUpdatePacket()
	update.AddUpdate(*sess)
	update.AddUpdate(*newScreen)
	if activate {
		update.AddUpdate(session.ActiveSessionIdUpdate(newSessionId))
	}
	return update, nil
}

func SwitchScreenById(ctx context.Context, sessionId string, screenId string) (*scbus.ModelUpdatePacketType, error) {
	session.SetActiveSessionId(ctx, sessionId)
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT screenid FROM screen WHERE sessionid = ? AND screenid = ?`
		if !tx.Exists(query, sessionId, screenId) {
			return fmt.Errorf("cannot switch to screen, screen=%s does not exist in session=%s", screenId, sessionId)
		}
		query = `UPDATE session SET activescreenid = ? WHERE sessionid = ?`
		tx.Exec(query, screenId, sessionId)
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	bareSession, err := session.GetBareSessionById(ctx, sessionId)
	if err != nil {
		return nil, err
	}
	update := scbus.MakeUpdatePacket()
	update.AddUpdate(ActiveSessionIdUpdate(sessionId))
	update.AddUpdate(*bareSession)
	memState := GetScreenMemState(screenId)
	if memState != nil {
		update.AddUpdate(CmdLineUpdate(memState.CmdInputText))
		UpdateWithCurrentOpenAICmdInfoChat(screenId, update)

		// Clear any previous status indicator for this screen
		err := ResetStatusIndicator_Update(update, screenId)
		if err != nil {
			// This is not a fatal error, so just log it
			log.Printf("error resetting status indicator when switching screens: %v\n", err)
		}
	}
	return update, nil
}
