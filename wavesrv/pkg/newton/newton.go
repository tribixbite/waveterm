package newton

import (
	"fmt"
	"strings"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/utilfn"
	"mvdan.cc/sh/v3/syntax"
)

func TraverseCmds(cmdStr utilfn.StrWithPos) error {
	cmdReader := strings.NewReader(cmdStr.String())
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := parser.Parse(cmdReader, "cmd")
	if err != nil {
		return fmt.Errorf("error parsing command: %w", err)
	}
	lastCmd := file.Last
	fmt.Printf("lastCmd: %#v\n", lastCmd)
	return nil
}
