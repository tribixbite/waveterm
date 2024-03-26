package line

import (
	"context"
	"time"

	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbase"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
)

const MaxLineStateSize = 4 * 1024 // 4k for now, can raise if needed
const LineNoHeight = -1

const (
	LineTypeCmd    = "cmd"
	LineTypeText   = "text"
	LineTypeOpenAI = "openai"
)

const (
	LineState_Source   = "prompt:source"
	LineState_File     = "prompt:file"
	LineState_FileUrl  = "wave:fileurl"
	LineState_Min      = "wave:min"
	LineState_Template = "template"
	LineState_Mode     = "mode"
	LineState_Lang     = "lang"
	LineState_Minimap  = "minimap"
)

type LineType struct {
	ScreenId      string         `json:"screenid"`
	UserId        string         `json:"userid"`
	LineId        string         `json:"lineid"`
	Ts            int64          `json:"ts"`
	LineNum       int64          `json:"linenum"`
	LineNumTemp   bool           `json:"linenumtemp,omitempty"`
	LineLocal     bool           `json:"linelocal"`
	LineType      string         `json:"linetype"`
	LineState     map[string]any `json:"linestate"`
	Renderer      string         `json:"renderer,omitempty"`
	Text          string         `json:"text,omitempty"`
	Ephemeral     bool           `json:"ephemeral,omitempty"`
	ContentHeight int64          `json:"contentheight,omitempty"`
	Star          bool           `json:"star,omitempty"`
	Archived      bool           `json:"archived,omitempty"`
	Remove        bool           `json:"remove,omitempty"`
}

func (LineType) UseDBMap() {}

type ScreenLinesType struct {
	ScreenId string            `json:"screenid"`
	Lines    []*LineType       `json:"lines" dbmap:"-"`
	Cmds     []*sstore.CmdType `json:"cmds" dbmap:"-"`
}

func (ScreenLinesType) UseDBMap() {}

func (ScreenLinesType) GetType() string {
	return "screenlines"
}

func makeNewLineCmd(screenId string, userId string, lineId string, renderer string, lineState map[string]any) *LineType {
	rtn := &LineType{}
	rtn.ScreenId = screenId
	rtn.UserId = userId
	rtn.LineId = lineId
	rtn.Ts = time.Now().UnixMilli()
	rtn.LineLocal = true
	rtn.LineType = LineTypeCmd
	rtn.LineId = lineId
	rtn.ContentHeight = LineNoHeight
	rtn.Renderer = renderer
	if lineState == nil {
		lineState = make(map[string]any)
	}
	rtn.LineState = lineState
	return rtn
}

func makeNewLineText(screenId string, userId string, text string) *LineType {
	rtn := &LineType{}
	rtn.ScreenId = screenId
	rtn.UserId = userId
	rtn.LineId = scbase.GenWaveUUID()
	rtn.Ts = time.Now().UnixMilli()
	rtn.LineLocal = true
	rtn.LineType = LineTypeText
	rtn.Text = text
	rtn.ContentHeight = LineNoHeight
	rtn.LineState = make(map[string]any)
	return rtn
}

func makeNewLineOpenAI(screenId string, userId string, lineId string) *LineType {
	rtn := &LineType{}
	rtn.ScreenId = screenId
	rtn.UserId = userId
	rtn.LineId = lineId
	rtn.Ts = time.Now().UnixMilli()
	rtn.LineLocal = true
	rtn.LineType = LineTypeOpenAI
	rtn.ContentHeight = LineNoHeight
	rtn.Renderer = sstore.CmdRendererOpenAI
	rtn.LineState = make(map[string]any)
	return rtn
}

func AddCommentLine(ctx context.Context, screenId string, userId string, commentText string) (*LineType, error) {
	rtnLine := makeNewLineText(screenId, userId, commentText)
	err := InsertLine(ctx, rtnLine, nil)
	if err != nil {
		return nil, err
	}
	return rtnLine, nil
}

func AddOpenAILine(ctx context.Context, screenId string, userId string, cmd *sstore.CmdType) (*LineType, error) {
	rtnLine := makeNewLineOpenAI(screenId, userId, cmd.LineId)
	err := InsertLine(ctx, rtnLine, cmd)
	if err != nil {
		return nil, err
	}
	return rtnLine, nil
}

func AddCmdLine(ctx context.Context, screenId string, userId string, cmd *sstore.CmdType, renderer string, lineState map[string]any) (*LineType, error) {
	rtnLine := makeNewLineCmd(screenId, userId, cmd.LineId, renderer, lineState)
	err := InsertLine(ctx, rtnLine, cmd)
	if err != nil {
		return nil, err
	}
	return rtnLine, nil
}
