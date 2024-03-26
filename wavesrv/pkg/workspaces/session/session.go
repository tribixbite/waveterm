package session

import "github.com/wavetermdev/waveterm/wavesrv/pkg/sstore"

type SessionType struct {
	SessionId      string                   `json:"sessionid"`
	Name           string                   `json:"name"`
	SessionIdx     int64                    `json:"sessionidx"`
	ActiveScreenId string                   `json:"activescreenid"`
	ShareMode      string                   `json:"sharemode"`
	NotifyNum      int64                    `json:"notifynum"`
	Archived       bool                     `json:"archived,omitempty"`
	ArchivedTs     int64                    `json:"archivedts,omitempty"`
	Remotes        []*sstore.RemoteInstance `json:"remotes"`

	// only for updates
	Remove bool `json:"remove,omitempty"`
}

func (SessionType) GetType() string {
	return "session"
}

func MakeSessionUpdateForRemote(sessionId string, ri *sstore.RemoteInstance) SessionType {
	return SessionType{
		SessionId: sessionId,
		Remotes:   []*sstore.RemoteInstance{ri},
	}
}

type SessionTombstoneType struct {
	SessionId string `json:"sessionid"`
	Name      string `json:"name"`
	DeletedTs int64  `json:"deletedts"`
}

func (SessionTombstoneType) UseDBMap() {}

func (SessionTombstoneType) GetType() string {
	return "sessiontombstone"
}

type SessionStatsType struct {
	SessionId          string              `json:"sessionid"`
	NumScreens         int                 `json:"numscreens"`
	NumArchivedScreens int                 `json:"numarchivedscreens"`
	NumLines           int                 `json:"numlines"`
	NumCmds            int                 `json:"numcmds"`
	DiskStats          SessionDiskSizeType `json:"diskstats"`
}
