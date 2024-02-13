// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package sstore

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wavetermdev/waveterm/waveshell/pkg/cirfile"
	"github.com/wavetermdev/waveterm/waveshell/pkg/shexec"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbase"
)

const MaxDBFileSize = 10 * 1024

var screenDirLock = &sync.Mutex{}
var screenDirCache = make(map[string]string) // locked with screenDirLock

var globalDBFileCache = makeDBFileCache()

type dbFileCacheEntry struct {
	DBLock *sync.Mutex
	DB     *sqlx.DB
	InUse  atomic.Bool
}

type DBFileCache struct {
	Lock  *sync.Mutex
	Cache map[string]*dbFileCacheEntry
}

func makeDBFileCache() *DBFileCache {
	return &DBFileCache{
		Lock:  &sync.Mutex{},
		Cache: make(map[string]*dbFileCacheEntry),
	}
}

func (c *DBFileCache) GetDB(screenId string) (*sqlx.DB, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	entry := c.Cache[screenId]
	if entry != nil {
		entry.DBLock.Lock()
		entry.InUse.Store(true)
		return entry.DB, nil
	}
	_, err := EnsureScreenDir(screenId)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *DBFileCache) ReleaseDB(screenId string, db *sqlx.DB) {
	entry := c.Cache[screenId]
	if entry == nil {
		// this shouldn't happen (error)
		log.Printf("[db] error missing cache entry for dbfile %s", screenId)
		return
	}
	entry.DBLock.Unlock()
	entry.InUse.Store(false)
	// noop for now
}

// fulfills the txwrap DBGetter interface
type DBFileGetter struct {
	ScreenId string
}

func (g DBFileGetter) GetDB(ctx context.Context) (*sqlx.DB, error) {
	return globalDBFileCache.GetDB(g.ScreenId)
}

func (g DBFileGetter) ReleaseDB(db *sqlx.DB) {
	globalDBFileCache.ReleaseDB(g.ScreenId, db)
}

func TryConvertPtyFile(ctx context.Context, screenId string, lineId string) error {
	stat, err := StatCmdPtyFile(ctx, screenId, lineId)
	if err != nil {
		return fmt.Errorf("convert ptyfile, cannot stat: %w", err)
	}
	if stat.DataSize > MaxDBFileSize {
		return nil
	}
	return nil
}

func CreateCmdPtyFile(ctx context.Context, screenId string, lineId string, maxSize int64) error {
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return err
	}
	f, err := cirfile.CreateCirFile(ptyOutFileName, maxSize)
	if err != nil {
		return err
	}
	return f.Close()
}

func StatCmdPtyFile(ctx context.Context, screenId string, lineId string) (*cirfile.Stat, error) {
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return nil, err
	}
	return cirfile.StatCirFile(ctx, ptyOutFileName)
}

func ClearCmdPtyFile(ctx context.Context, screenId string, lineId string) error {
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return err
	}
	stat, err := cirfile.StatCirFile(ctx, ptyOutFileName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	os.Remove(ptyOutFileName) // ignore error
	var maxSize int64 = shexec.DefaultMaxPtySize
	if stat != nil {
		maxSize = stat.MaxSize
	}
	err = CreateCmdPtyFile(ctx, screenId, lineId, maxSize)
	if err != nil {
		return err
	}
	return nil
}

func AppendToCmdPtyBlob(ctx context.Context, screenId string, lineId string, data []byte, pos int64) (*PtyDataUpdate, error) {
	if screenId == "" {
		return nil, fmt.Errorf("cannot append to PtyBlob, screenid is not set")
	}
	if pos < 0 {
		return nil, fmt.Errorf("invalid seek pos '%d' in AppendToCmdPtyBlob", pos)
	}
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return nil, err
	}
	f, err := cirfile.OpenCirFile(ptyOutFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	err = f.WriteAt(ctx, data, pos)
	if err != nil {
		return nil, err
	}
	data64 := base64.StdEncoding.EncodeToString(data)
	update := &PtyDataUpdate{
		ScreenId:   screenId,
		LineId:     lineId,
		PtyPos:     pos,
		PtyData64:  data64,
		PtyDataLen: int64(len(data)),
	}
	err = MaybeInsertPtyPosUpdate(ctx, screenId, lineId)
	if err != nil {
		// just log
		log.Printf("error inserting ptypos update %s/%s: %v\n", screenId, lineId, err)
	}
	return update, nil
}

// returns (real-offset, data, err)
func ReadFullPtyOutFile(ctx context.Context, screenId string, lineId string) (int64, []byte, error) {
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return 0, nil, err
	}
	f, err := cirfile.OpenCirFile(ptyOutFileName)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()
	return f.ReadAll(ctx)
}

// returns (real-offset, data, err)
func ReadPtyOutFile(ctx context.Context, screenId string, lineId string, offset int64, maxSize int64) (int64, []byte, error) {
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return 0, nil, err
	}
	f, err := cirfile.OpenCirFile(ptyOutFileName)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()
	return f.ReadAtWithMax(ctx, offset, maxSize)
}

type SessionDiskSizeType struct {
	NumFiles   int
	TotalSize  int64
	ErrorCount int
	Location   string
}

func directorySize(dirName string) (SessionDiskSizeType, error) {
	var rtn SessionDiskSizeType
	rtn.Location = dirName
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return rtn, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			rtn.ErrorCount++
			continue
		}
		finfo, err := entry.Info()
		if err != nil {
			rtn.ErrorCount++
			continue
		}
		rtn.NumFiles++
		rtn.TotalSize += finfo.Size()
	}
	return rtn, nil
}

func SessionDiskSize(sessionId string) (SessionDiskSizeType, error) {
	sessionDir, err := scbase.EnsureSessionDir(sessionId)
	if err != nil {
		return SessionDiskSizeType{}, err
	}
	return directorySize(sessionDir)
}

func FullSessionDiskSize() (map[string]SessionDiskSizeType, error) {
	sdir := scbase.GetSessionsDir()
	entries, err := os.ReadDir(sdir)
	if err != nil {
		return nil, err
	}
	rtn := make(map[string]SessionDiskSizeType)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		_, err = uuid.Parse(name)
		if err != nil {
			continue
		}
		diskSize, err := directorySize(path.Join(sdir, name))
		if err != nil {
			continue
		}
		rtn[name] = diskSize
	}
	return rtn, nil
}

func DeletePtyOutFile(ctx context.Context, screenId string, lineId string) error {
	ptyOutFileName, err := PtyOutFile(screenId, lineId)
	if err != nil {
		return err
	}
	err = os.Remove(ptyOutFileName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func GoDeleteScreenDirs(screenIds ...string) {
	go func() {
		for _, screenId := range screenIds {
			deleteScreenDirMakeCtx(screenId)
		}
	}()
}

func deleteScreenDirMakeCtx(screenId string) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()
	err := DeleteScreenDir(ctx, screenId)
	if err != nil {
		log.Printf("error deleting screendir %s: %v\n", screenId, err)
	}
}

func DeleteScreenDir(ctx context.Context, screenId string) error {
	screenDir, err := EnsureScreenDir(screenId)
	if err != nil {
		return fmt.Errorf("error getting screendir: %w", err)
	}
	log.Printf("delete screen dir, remove-all %s\n", screenDir)
	return os.RemoveAll(screenDir)
}

func EnsureScreenDir(screenId string) (string, error) {
	if screenId == "" {
		return "", fmt.Errorf("cannot get screen dir for blank sessionid")
	}
	screenDirLock.Lock()
	sdir, ok := screenDirCache[screenId]
	screenDirLock.Unlock()
	if ok {
		return sdir, nil
	}
	scHome := scbase.GetWaveHomeDir()
	sdir = path.Join(scHome, scbase.ScreensDirBaseName, screenId)
	err := scbase.EnsureDir(sdir)
	if err != nil {
		return "", err
	}
	screenDirLock.Lock()
	screenDirCache[screenId] = sdir
	screenDirLock.Unlock()
	return sdir, nil
}

func PtyOutFile(screenId string, lineId string) (string, error) {
	sdir, err := EnsureScreenDir(screenId)
	if err != nil {
		return "", err
	}
	if screenId == "" {
		return "", fmt.Errorf("cannot get ptyout file for blank screenid")
	}
	if lineId == "" {
		return "", fmt.Errorf("cannot get ptyout file for blank lineid")
	}
	return fmt.Sprintf("%s/%s.ptyout.cf", sdir, lineId), nil
}
