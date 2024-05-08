// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package blockstore

import "sync"

type cacheKey struct {
	BlockId string
	Name    string
}

var blockstoreCache map[cacheKey]*CacheEntry = make(map[cacheKey]*CacheEntry)
var globalLock *sync.Mutex = &sync.Mutex{}
