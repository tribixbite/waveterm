package newton

import (
	"testing"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/utilfn"
)

func TestNewtonParse(t *testing.T) {
	testStr := utilfn.StrWithPos{
		Str: `echo "hello world" | cat -n | grep "hello" | sed 's/hello/hi/g' > /tmp/test.txt`,
		Pos: 1,
	}
	err := TraverseCmds(testStr)
	t.Error()
	if err != nil {
		t.Error(err)
	}
}
