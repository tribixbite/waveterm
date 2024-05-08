// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package blockstore

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// this flush timer is so complicated because of testing
type flushTimer struct {
	T         *time.Ticker
	IsRunning bool
	DoneCh    chan struct{}
	CVar      *sync.Cond
}

var globalFlushTimer *flushTimer = &flushTimer{T: nil, IsRunning: false, CVar: sync.NewCond(&sync.Mutex{})}

// stops the flush timer if running -- will block until the timer is stopped
// needed for testing :/
func stopFlushTimer() {
	globalFlushTimer.CVar.L.Lock()
	defer globalFlushTimer.CVar.L.Unlock()
	if !globalFlushTimer.IsRunning {
		return
	}
	close(globalFlushTimer.DoneCh)
	for globalFlushTimer.IsRunning {
		globalFlushTimer.CVar.Wait()
	}
}

func _createFlushTimer(flushTimeout time.Duration) bool {
	globalFlushTimer.CVar.L.Lock()
	defer globalFlushTimer.CVar.L.Unlock()
	if globalFlushTimer.IsRunning {
		return false
	}
	globalFlushTimer.T = time.NewTicker(flushTimeout)
	globalFlushTimer.DoneCh = make(chan struct{})
	globalFlushTimer.IsRunning = true
	return true
}

// starts the flush timer with given timeout (in a go routine).  if timer is already running it will return an error
// needed for testing :/
func startFlushTimer(flushTimeout time.Duration) error {
	created := _createFlushTimer(flushTimeout)
	if !created {
		return fmt.Errorf("flush timer already running")
	}
	go func() {
		defer func() {
			globalFlushTimer.CVar.L.Lock()
			defer globalFlushTimer.CVar.L.Unlock()
			globalFlushTimer.T.Stop()
			globalFlushTimer.IsRunning = false
			globalFlushTimer.CVar.Broadcast()
		}()
		for {
			select {
			case <-globalFlushTimer.T.C:
				FlushCache(context.Background())
			case <-globalFlushTimer.DoneCh:
				return
			}
		}
	}()
	return nil
}
