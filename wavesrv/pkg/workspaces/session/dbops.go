package session

import (
	"context"
	"fmt"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
)

// includes archived sessions
func GetBareSessions(ctx context.Context) ([]*SessionType, error) {
	var rtn []*SessionType
	err := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT * FROM session ORDER BY archived, sessionidx, archivedts`
		tx.Select(&rtn, query)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

// does not include archived, finds lowest sessionidx (for resetting active session)
func GetFirstSessionId(ctx context.Context) (string, error) {
	var rtn []string
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT sessionid from session WHERE NOT archived ORDER by sessionidx`
		rtn = tx.SelectStrings(query)
		return nil
	})
	if txErr != nil {
		return "", txErr
	}
	if len(rtn) == 0 {
		return "", nil
	}
	return rtn[0], nil
}

func GetBareSessionById(ctx context.Context, sessionId string) (*SessionType, error) {
	var rtn SessionType
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT * FROM session WHERE sessionid = ?`
		tx.Get(&rtn, query, sessionId)
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	if rtn.SessionId == "" {
		return nil, nil
	}
	return &rtn, nil
}

const GetAllSessionsQuery = `SELECT * FROM session ORDER BY archived, sessionidx, archivedts`

// Gets all sessions, including archived
func GetAllSessions(ctx context.Context) ([]*SessionType, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) ([]*SessionType, error) {
		rtn := []*SessionType{}
		tx.Select(&rtn, GetAllSessionsQuery)
		return rtn, nil
	})
}

func GetSessionById(ctx context.Context, id string) (*SessionType, error) {
	allSessions, err := GetAllSessions(ctx)
	if err != nil {
		return nil, err
	}
	for _, session := range allSessions {
		if session.SessionId == id {
			return session, nil
		}
	}
	return nil, nil
}

// counts non-archived sessions
func GetSessionCount(ctx context.Context) (int, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (int, error) {
		query := `SELECT COALESCE(count(*), 0) FROM session WHERE NOT archived`
		numSessions := tx.GetInt(query)
		return numSessions, nil
	})
}

func GetSessionByName(ctx context.Context, name string) (*SessionType, error) {
	var session *SessionType
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT sessionid FROM session WHERE name = ?`
		sessionId := tx.GetString(query, name)
		if sessionId == "" {
			return nil
		}
		var err error
		session, err = GetSessionById(tx.Context(), sessionId)
		if err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return session, nil
}

func SetActiveSessionId(ctx context.Context, sessionId string) error {
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT sessionid FROM session WHERE sessionid = ?`
		if !tx.Exists(query, sessionId) {
			return fmt.Errorf("cannot switch to session, not found")
		}
		query = `UPDATE client SET activesessionid = ?`
		tx.Exec(query, sessionId)
		return nil
	})
	return txErr
}

func GetActiveSessionId(ctx context.Context) (string, error) {
	var rtnId string
	txErr := sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT activesessionid FROM client`
		rtnId = tx.GetString(query)
		return nil
	})
	return rtnId, txErr
}
