package screen

import (
	"github.com/wavetermdev/waveterm/wavesrv/pkg/dbutil"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbus"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"
)

const (
	ScreenFocusInput = "input"
	ScreenFocusCmd   = "cmd"
)

type ScreenOptsType struct {
	TabColor string `json:"tabcolor,omitempty"`
	TabIcon  string `json:"tabicon,omitempty"`
	PTerm    string `json:"pterm,omitempty"`
}

type ScreenLinesType struct {
	ScreenId string             `json:"screenid"`
	Lines    []*sstore.LineType `json:"lines" dbmap:"-"`
	Cmds     []*sstore.CmdType  `json:"cmds" dbmap:"-"`
}

func (ScreenLinesType) UseDBMap() {}

func (ScreenLinesType) GetType() string {
	return "screenlines"
}

type ScreenWebShareOpts struct {
	ShareName string `json:"sharename"`
	ViewKey   string `json:"viewkey"`
}

type ScreenCreateOpts struct {
	BaseScreenId string
	CopyRemote   bool
	CopyCwd      bool
	CopyEnv      bool
}

func (sco ScreenCreateOpts) HasCopy() bool {
	return sco.CopyRemote || sco.CopyCwd || sco.CopyEnv
}

type ScreenSidebarOptsType struct {
	Open  bool   `json:"open,omitempty"`
	Width string `json:"width,omitempty"`

	// this used to be more complicated (sections with types).  simplified for this release
	SidebarLineId string `json:"sidebarlineid,omitempty"`
}

type ScreenViewOptsType struct {
	Sidebar *ScreenSidebarOptsType `json:"sidebar,omitempty"`
}

type ScreenAnchorType struct {
	AnchorLine   int `json:"anchorline,omitempty"`
	AnchorOffset int `json:"anchoroffset,omitempty"`
}

type ScreenType struct {
	SessionId      string               `json:"sessionid"`
	ScreenId       string               `json:"screenid"`
	Name           string               `json:"name"`
	ScreenIdx      int64                `json:"screenidx"`
	ScreenOpts     ScreenOptsType       `json:"screenopts"`
	ScreenViewOpts ScreenViewOptsType   `json:"screenviewopts"`
	OwnerId        string               `json:"ownerid"`
	ShareMode      string               `json:"sharemode"`
	WebShareOpts   *ScreenWebShareOpts  `json:"webshareopts,omitempty"`
	CurRemote      sstore.RemotePtrType `json:"curremote"`
	NextLineNum    int64                `json:"nextlinenum"`
	SelectedLine   int64                `json:"selectedline"`
	Anchor         ScreenAnchorType     `json:"anchor"`
	FocusType      string               `json:"focustype"`
	Archived       bool                 `json:"archived,omitempty"`
	ArchivedTs     int64                `json:"archivedts,omitempty"`

	// only for updates
	Remove bool `json:"remove,omitempty"`
}

func (s *ScreenType) ToMap() map[string]interface{} {
	rtn := make(map[string]interface{})
	rtn["sessionid"] = s.SessionId
	rtn["screenid"] = s.ScreenId
	rtn["name"] = s.Name
	rtn["screenidx"] = s.ScreenIdx
	rtn["screenopts"] = dbutil.QuickJson(s.ScreenOpts)
	rtn["screenviewopts"] = dbutil.QuickJson(s.ScreenViewOpts)
	rtn["ownerid"] = s.OwnerId
	rtn["sharemode"] = s.ShareMode
	rtn["webshareopts"] = dbutil.QuickNullableJson(s.WebShareOpts)
	rtn["curremoteownerid"] = s.CurRemote.OwnerId
	rtn["curremoteid"] = s.CurRemote.RemoteId
	rtn["curremotename"] = s.CurRemote.Name
	rtn["nextlinenum"] = s.NextLineNum
	rtn["selectedline"] = s.SelectedLine
	rtn["anchor"] = dbutil.QuickJson(s.Anchor)
	rtn["focustype"] = s.FocusType
	rtn["archived"] = s.Archived
	rtn["archivedts"] = s.ArchivedTs
	return rtn
}

func (s *ScreenType) FromMap(m map[string]interface{}) bool {
	dbutil.QuickSetStr(&s.SessionId, m, "sessionid")
	dbutil.QuickSetStr(&s.ScreenId, m, "screenid")
	dbutil.QuickSetStr(&s.Name, m, "name")
	dbutil.QuickSetInt64(&s.ScreenIdx, m, "screenidx")
	dbutil.QuickSetJson(&s.ScreenOpts, m, "screenopts")
	dbutil.QuickSetJson(&s.ScreenViewOpts, m, "screenviewopts")
	dbutil.QuickSetStr(&s.OwnerId, m, "ownerid")
	dbutil.QuickSetStr(&s.ShareMode, m, "sharemode")
	dbutil.QuickSetNullableJson(&s.WebShareOpts, m, "webshareopts")
	dbutil.QuickSetStr(&s.CurRemote.OwnerId, m, "curremoteownerid")
	dbutil.QuickSetStr(&s.CurRemote.RemoteId, m, "curremoteid")
	dbutil.QuickSetStr(&s.CurRemote.Name, m, "curremotename")
	dbutil.QuickSetInt64(&s.NextLineNum, m, "nextlinenum")
	dbutil.QuickSetInt64(&s.SelectedLine, m, "selectedline")
	dbutil.QuickSetJson(&s.Anchor, m, "anchor")
	dbutil.QuickSetStr(&s.FocusType, m, "focustype")
	dbutil.QuickSetBool(&s.Archived, m, "archived")
	dbutil.QuickSetInt64(&s.ArchivedTs, m, "archivedts")
	return true
}

func (ScreenType) GetType() string {
	return "screen"
}

func AddScreenUpdate(update *scbus.ModelUpdatePacketType, newScreen *ScreenType) {
	if newScreen == nil {
		return
	}
	screenUpdates := scbus.GetUpdateItems[ScreenType](update)
	for _, screenUpdate := range screenUpdates {
		if screenUpdate.ScreenId == newScreen.ScreenId {
			screenUpdate = newScreen
			return
		}
	}
	update.AddUpdate(newScreen)
}

type ScreenTombstoneType struct {
	ScreenId   string         `json:"screenid"`
	SessionId  string         `json:"sessionid"`
	Name       string         `json:"name"`
	DeletedTs  int64          `json:"deletedts"`
	ScreenOpts ScreenOptsType `json:"screenopts"`
}

func (ScreenTombstoneType) UseDBMap() {}

func (ScreenTombstoneType) GetType() string {
	return "screentombstone"
}
