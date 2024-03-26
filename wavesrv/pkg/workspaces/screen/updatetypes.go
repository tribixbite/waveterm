package screen

type ScreenUpdateType struct {
	UpdateId   int64  `json:"updateid"`
	ScreenId   string `json:"screenid"`
	LineId     string `json:"lineid"`
	UpdateType string `json:"updatetype"`
	UpdateTs   int64  `json:"updatets"`
}

func (ScreenUpdateType) UseDBMap() {}
