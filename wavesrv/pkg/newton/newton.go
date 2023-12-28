package newton

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/utilfn"
	"mvdan.cc/sh/v3/syntax"
)

// Recurse through a syntax.Stmt and find the last statement that wraps an expression.
func findLastStmtWithExpr(stmt *syntax.Stmt) *syntax.Stmt {
	switch x := stmt.Cmd.(type) {
	case *syntax.BinaryCmd:
		if x.Y != nil {
			return findLastStmtWithExpr(x.Y)
		} else {
			return findLastStmtWithExpr(x.X)
		}
	case *syntax.CallExpr:
		return stmt
	default:
		return nil
	}
}

// Parse a syntax.Word into a complete string.
func getWordStr(word *syntax.Word) string {
	wordBuf := new(bytes.Buffer)
	for _, part := range word.Parts {
		switch x := part.(type) {
		case *syntax.Lit:
			wordBuf.WriteString(x.Value)
		case *syntax.SglQuoted:
			wordBuf.WriteString(fmt.Sprintf("'%s'", x.Value))
		case *syntax.DblQuoted:
			wordBuf.WriteByte('"')
			for _, part := range x.Parts {
				switch x := part.(type) {
				case *syntax.Lit:
					wordBuf.WriteString(x.Value)
				case *syntax.ParamExp:
					wordBuf.WriteString(x.Param.Value)
				}
			}
			wordBuf.WriteByte('"')
		}
	}
	return wordBuf.String()
}

func TraverseCmds(cmdStr utilfn.StrWithPos) error {
	if cmdStr.Str == "" {
		return nil
	}
	cmdReader := strings.NewReader(cmdStr.Str)
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := parser.Parse(cmdReader, "")
	if err != nil {
		return fmt.Errorf("error parsing command: %w", err)
	}

	// syntax.DebugPrint(os.Stdout, file)

	// Take the last statement and recurse it to find the last statement that wraps an expression.
	lastStmt := findLastStmtWithExpr(file.Stmts[len(file.Stmts)-1])

	if lastStmt == nil {
		return fmt.Errorf("error finding last statement")
	}

	if lastStmt.Redirs != nil {
		fmt.Println("contains redirects, ignoring parsing")
		return nil
	}

	cmd := getWordStr(lastStmt.Cmd.(*syntax.CallExpr).Args[0])
	args := make([]string, len(lastStmt.Cmd.(*syntax.CallExpr).Args)-1)

	for i, arg := range lastStmt.Cmd.(*syntax.CallExpr).Args[1:] {
		args[i] = getWordStr(arg)
	}

	fmt.Printf("cmd: %s\n", cmd)
	fmt.Printf("args: %v\n", args)

	return nil
}
