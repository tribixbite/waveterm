package line

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/dbutil"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
)

// special "E" returns last unarchived line, "EA" returns last line (even if archived)
func FindLineIdByArg(ctx context.Context, screenId string, lineArg string) (string, error) {
	return sstore.WithTxRtn(ctx, func(tx *sstore.TxWrap) (string, error) {
		if lineArg == "E" {
			query := `SELECT lineid FROM line WHERE screenid = ? AND NOT archived ORDER BY linenum DESC LIMIT 1`
			lineId := tx.GetString(query, screenId)
			return lineId, nil
		}
		if lineArg == "EA" {
			query := `SELECT lineid FROM line WHERE screenid = ? ORDER BY linenum DESC LIMIT 1`
			lineId := tx.GetString(query, screenId)
			return lineId, nil
		}
		lineNum, err := strconv.Atoi(lineArg)
		if err == nil {
			// valid linenum
			query := `SELECT lineid FROM line WHERE screenid = ? AND linenum = ?`
			lineId := tx.GetString(query, screenId, lineNum)
			return lineId, nil
		} else if len(lineArg) == 8 {
			// prefix id string match
			query := `SELECT lineid FROM line WHERE screenid = ? AND substr(lineid, 1, 8) = ?`
			lineId := tx.GetString(query, screenId, lineArg)
			return lineId, nil
		} else {
			// id match
			query := `SELECT lineid FROM line WHERE screenid = ? AND lineid = ?`
			lineId := tx.GetString(query, screenId, lineArg)
			return lineId, nil
		}
	})
}

func GetLineCmdByLineId(ctx context.Context, screenId string, lineId string) (*LineType, *sstore.CmdType, error) {
	return sstore.WithTxRtn3(ctx, func(tx *sstore.TxWrap) (*LineType, *sstore.CmdType, error) {
		query := `SELECT * FROM line WHERE screenid = ? AND lineid = ?`
		lineVal := dbutil.GetMappable[*LineType](tx, query, screenId, lineId)
		if lineVal == nil {
			return nil, nil, nil
		}
		var cmdRtn *sstore.CmdType
		query = `SELECT * FROM cmd WHERE screenid = ? AND lineid = ?`
		cmdRtn = dbutil.GetMapGen[*sstore.CmdType](tx, query, screenId, lineId)
		return lineVal, cmdRtn, nil
	})
}

func InsertLine(ctx context.Context, line *LineType, cmd *sstore.CmdType) error {
	if line == nil {
		return fmt.Errorf("line cannot be nil")
	}
	if line.LineId == "" {
		return fmt.Errorf("line must have lineid set")
	}
	if line.LineNum != 0 {
		return fmt.Errorf("line should not hage linenum set")
	}
	if cmd != nil && cmd.ScreenId == "" {
		return fmt.Errorf("cmd should have screenid set")
	}
	qjs := dbutil.QuickJson(line.LineState)
	if len(qjs) > MaxLineStateSize {
		return fmt.Errorf("linestate exceeds maxsize, size[%d] max[%d]", len(qjs), MaxLineStateSize)
	}
	return sstore.WithTx(ctx, func(tx *sstore.TxWrap) error {
		query := `SELECT screenid FROM screen WHERE screenid = ?`
		if !tx.Exists(query, line.ScreenId) {
			return fmt.Errorf("screen not found, cannot insert line[%s]", line.ScreenId)
		}
		query = `SELECT nextlinenum FROM screen WHERE screenid = ?`
		nextLineNum := tx.GetInt(query, line.ScreenId)
		line.LineNum = int64(nextLineNum)
		query = `INSERT INTO line  ( screenid, userid, lineid, ts, linenum, linenumtemp, linelocal, linetype, linestate, text, renderer, ephemeral, contentheight, star, archived)
                            VALUES (:screenid,:userid,:lineid,:ts,:linenum,:linenumtemp,:linelocal,:linetype,:linestate,:text,:renderer,:ephemeral,:contentheight,:star,:archived)`
		tx.NamedExec(query, dbutil.ToDBMap(line, false))
		query = `UPDATE screen SET nextlinenum = ? WHERE screenid = ?`
		tx.Exec(query, nextLineNum+1, line.ScreenId)
		if cmd != nil {
			cmd.OrigTermOpts = cmd.TermOpts
			cmdMap := cmd.ToMap()
			query = `
INSERT INTO cmd  ( screenid, lineid, remoteownerid, remoteid, remotename, cmdstr, rawcmdstr, festate, statebasehash, statediffhasharr, termopts, origtermopts, status, cmdpid, remotepid, donets, restartts, exitcode, durationms, rtnstate, runout, rtnbasehash, rtndiffhasharr)
          VALUES (:screenid,:lineid,:remoteownerid,:remoteid,:remotename,:cmdstr,:rawcmdstr,:festate,:statebasehash,:statediffhasharr,:termopts,:origtermopts,:status,:cmdpid,:remotepid,:donets,:restartts,:exitcode,:durationms,:rtnstate,:runout,:rtnbasehash,:rtndiffhasharr)
`
			tx.NamedExec(query, cmdMap)
		}
		if sstore.IsWebShare(tx, line.ScreenId) {
			sstore.InsertScreenLineUpdate(tx, line.ScreenId, line.LineId, sstore.UpdateType_LineNew)
		}
		return nil
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
		screen.Lines = dbutil.SelectMappable[*LineType](tx, query, screen.ScreenId)
		query = `SELECT * FROM cmd WHERE screenid = ?`
		screen.Cmds = dbutil.SelectMapsGen[*sstore.CmdType](tx, query, screen.ScreenId)
		return screen, nil
	})
}
