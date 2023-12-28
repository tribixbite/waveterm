package newton

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/utilfn"
	"mvdan.cc/sh/v3/syntax"
)

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
	// syntax.Walk(file, func(node syntax.Node) bool {
	// 	switch x := node.(type) {
	// 	case *syntax.ParamExp:
	// 		x.Param.Value = strings.ToUpper(x.Param.Value)
	// 	}
	// 	return true
	// })
	// syntax.NewPrinter().Print(os.Stdout, file)
	exprs := []*syntax.CallExpr{}
	stmts := []*syntax.Stmt{}
	debugStr := new(bytes.Buffer)
	syntax.DebugPrint(debugStr, file)
	fmt.Sprintln(debugStr.String())

	syntax.Walk(file, func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.CallExpr:
			exprs = append(exprs, x)
		case *syntax.Stmt:
			stmts = append(stmts, x)
		}
		return true
	})
	lastExpr := exprs[len(exprs)-1]
	lastExprStr := new(bytes.Buffer)
	for _, arg := range lastExpr.Args {
		lastExprStr.WriteString(arg.Lit())
	}

	lastStmt := stmts[len(stmts)-1]
	lastStmtStr := new(bytes.Buffer)
	for _, arg := range lastStmt.Cmd.(*syntax.CallExpr).Args {
		for _, part := range arg.Parts {
			switch x := part.(type) {
			case *syntax.Lit:
				lastStmtStr.WriteString(x.Value)
			case *syntax.SglQuoted:
				lastStmtStr.WriteString(fmt.Sprintf("'%s'", x.Value))
			case *syntax.DblQuoted:
				lastStmtStr.WriteByte('"')
				for _, part := range x.Parts {
					switch x := part.(type) {
					case *syntax.Lit:
						lastStmtStr.WriteString(x.Value)
					case *syntax.ParamExp:
						lastStmtStr.WriteString(x.Param.Value)
					}
				}
				lastStmtStr.WriteByte('"')
			}
		}
	}

	for _, redir := range lastStmt.Redirs {
		lastStmtStr.WriteString(redir.Op.String())
		lastStmtStr.WriteString(redir.Word.Lit())
	}
	fmt.Printf("last statement: %s\n", lastStmtStr.String())
	return nil
}
