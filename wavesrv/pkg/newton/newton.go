package newton

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-set/v2"
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

type CmdList []string
type CmdSet struct {
	// OnceValue gets the CmdSet singleton.
	get func() *set.Set[string]
}

const NewtonDir = "/Users/evan/source/newton/x"

// Initialize the CmdSet singleton.
func getCmdSet() CmdSet {
	return CmdSet{
		get: sync.OnceValue[*set.Set[string]](func() *set.Set[string] {
			var cmds CmdList
			err := utilfn.ReadJsonFile(filepath.Join(NewtonDir, "index.json"), &cmds)
			if err != nil {
				fmt.Printf("error reading json file: %v\n", err)
				return nil
			}
			return set.From[string](cmds)
		}),
	}
}

// Parses a command string to exract the last command and its arguments.
func parseCommand(cmdStr string) (cmd string, args []string, err error) {
	cmdReader := strings.NewReader(cmdStr)
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := parser.Parse(cmdReader, "")
	if err != nil {
		return "", nil, fmt.Errorf("error parsing command: %w", err)
	}

	// syntax.DebugPrint(os.Stdout, file)

	// Take the last statement and recurse it to find the last statement that wraps an expression.
	lastStmt := findLastStmtWithExpr(file.Stmts[len(file.Stmts)-1])

	if lastStmt == nil {
		return "", nil, fmt.Errorf("error finding last statement")
	}

	if lastStmt.Redirs != nil {
		fmt.Println("contains redirects, ignoring parsing")
		return "", nil, nil
	}

	cmd = getWordStr(lastStmt.Cmd.(*syntax.CallExpr).Args[0])
	args = make([]string, len(lastStmt.Cmd.(*syntax.CallExpr).Args)-1)

	for i, arg := range lastStmt.Cmd.(*syntax.CallExpr).Args[1:] {
		args[i] = getWordStr(arg)
	}

	return cmd, args, nil
}

type CmdSuggestion interface{}

func getCmdSuggestion(cmd string) (CmdSuggestion, error) {
	var cmdSuggestion CmdSuggestion
	err := utilfn.ReadJsonFile(filepath.Join(NewtonDir, fmt.Sprintf("%s.json", cmd)), &cmdSuggestion)
	if err != nil {
		return nil, fmt.Errorf("error reading json file: %v", err)
	}
	return cmdSuggestion, nil
}

// CmdSet singleton.
var cmdSet = getCmdSet()

// GetSuggestions takes a StrWithPos and returns autocomplete suggestions for the command.
func GetSuggestions(cmdStr utilfn.StrWithPos) error {
	if cmdStr.Str == "" {
		return nil
	}

	// cmd, args, err := parseCommand(cmdStr.Str)
	// if err != nil {
	// 	return fmt.Errorf("error parsing command: %w", err)
	// }

	// cmdAlt := ""
	// if len(args) > 0 {
	// 	cmdAlt = fmt.Sprintf("%s/%s", cmd, args[0])
	// }

	// fmt.Printf("cmd: %s\n", cmd)
	// fmt.Printf("args: %v\n", args)

	// if cmdSet.get().Contains(cmdAlt) {
	// 	// If the cmdAlt is available, use it and adjust the args.
	// 	fmt.Printf("cmdAlt \"%s\" is available\n", cmdAlt)
	// 	cmd = cmdAlt
	// 	args = args[1:]
	// } else if cmdSet.get().Contains(cmd) {
	// 	fmt.Printf("cmd \"%s\" available\n", cmd)
	// } else {
	// 	fmt.Printf("neither cmd \"%s\" nor cmdAlt \"%s\" are available\n", cmd, cmdAlt)
	// 	return nil
	// }

	// fmt.Printf("using cmd \"%s\" and args \"%v\"", cmd, args)
	// cmdSuggestion, err := getCmdSuggestion(cmd)
	// if err != nil {
	// 	return fmt.Errorf("error getting cmd suggestion: %w", err)
	// }

	// fmt.Printf("cmdSuggestion: %v\n", cmdSuggestion)
	cmdCobra = cobra.
		carapace.Gen(cmdStr.Str)

	return nil
}
