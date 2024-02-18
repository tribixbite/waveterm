// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package sstore

import "sync"

// This file contains structures related to holding display state for screens / UI

const (
	// ideal sizes for Sections/Zone
	// we allow these to drift a bit to deal with adding/removing lines
	DispZoneLines    = 16 * 16
	DispSectionLines = 16
	DispMaxRatio     = 1.6
)

// this will be a map at some point when we have multiple windows, singleton for now
var GlobalDisplayState *DispWindowInfo

// (row, px) <- we use this format so JS is
type DispHeight [2]int
type DispLineRange [2]int // [start, end] (inclusive).  set to [0,0] if no lines

// this struct comes from the FE
type DispWindowInfo struct {
	Lock            *sync.Mutex               `json:"-"`
	ActiveSessionId string                    `json:"activesessionid"`
	ActiveScreenId  string                    `json:"activescreenid"`
	Screens         map[string]DispScreenInfo `json:"screens"` // key is "sessionid:screenid" (not a struct for JS integration)
}

type DispScreenInfo struct {
	SessionId    string `json:"sessionid"`
	ScreenId     string `json:"screenid"`
	SelectedLine int    `json:"selectedline"`
	AnchorLine   int    `json:"anchorline"`
	AnchorOffset int    `json:"anchoroffset"`
}

type DispZoneInfo struct {
	ScreenId  string        `json:"screenid"`
	ZoneId    int           `json:"zoneid"`
	Height    DispHeight    `json:"height"`
	NumLines  int           `json:"numlines"`
	LineRange DispLineRange `json:"linerange"`
}

type DispSectionInfo struct {
	ScreenId  string         `json:"screenid"`
	ZoneId    int            `json:"zoneid"`
	SectionId int            `json:"sectionid"`
	Height    DispHeight     `json:"height"`
	LineRange DispLineRange  `json:"linerange"`
	Parts     []DispLineInfo `json:"parts"`
}

// includes separators.  combined into one struct for efficiency / JS transport
type DispLineInfo struct {
	// for separators
	IsSep     bool   `json:"issep,omitempty"`
	SepString string `json:"sepstring,omitempty"`

	// for lines
	LineId   string     `json:"lineid,omitempty"`
	LineNum  int        `json:"linenum,omitempty"`
	LineType string     `json:"linetype,omitempty"`
	Ts       int64      `json:"ts,omitempty"`
	Status   string     `json:"status,omitempty"`
	Renderer string     `json:"renderer,omitempty"`
	Prompt   string     `json:"prompt,omitempty"`
	CmdStr   string     `json:"cmdstr,omitempty"`
	Min      bool       `json:"min,omitempty"`
	TermOpts string     `json:"termopts,omitempty"`
	Height   DispHeight `json:"height,omitempty"`
}
