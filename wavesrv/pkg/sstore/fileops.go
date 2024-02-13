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
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/wavetermdev/waveterm/waveshell/pkg/cirfile"
	"github.com/wavetermdev/waveterm/waveshell/pkg/shexec"
	"github.com/wavetermdev/waveterm/wavesrv/pkg/scbase"
)

const MaxFileDBInlineFileSize = 10 * 1024

var screenDirLock = &sync.Mutex{}
var screenDirCache = make(map[string]string) // locked with screenDirLock

var globalFileDBCache = makeFileDBCache()

type fileDBCacheEntry struct {
	ScreenId string
	CVar     *sync.Cond // condition variable to lock entry fields and wait on InUse flag
	DB       *sqlx.DB   // can be nil (when not in use), will need to be reopend on access
	InUse    bool
	Migrated bool // only try to migrate the DB once per run
	Waiters  int
	LastUse  time.Time
	OpenErr  error // we cache open errors (and return them on GetDB)
}

// we store all screens in this cache (added on demand)
// when not in use we can close the DB object
type FileDBCache struct {
	Lock  *sync.Mutex
	Cache map[string]*fileDBCacheEntry // key = screenid
}

// will create an entry if it doesn't exist
func (dbc *FileDBCache) GetEntry(screenId string) *fileDBCacheEntry {
	dbc.Lock.Lock()
	defer dbc.Lock.Unlock()
	entry := dbc.Cache[screenId]
	if entry != nil {
		return entry
	}
	entry = &fileDBCacheEntry{
		ScreenId: screenId,
		CVar:     sync.NewCond(&sync.Mutex{}),
		DB:       nil,
		Migrated: false,
		InUse:    false,
		Waiters:  0,
		LastUse:  time.Time{},
	}
	dbc.Cache[screenId] = entry
	return entry
}

func makeFileDBCache() *FileDBCache {
	return &FileDBCache{
		Lock:  &sync.Mutex{},
		Cache: make(map[string]*fileDBCacheEntry),
	}
}

func MakeFileDBUrl(screenId string) (string, error) {
	screenDir, err := EnsureScreenDir(screenId)
	if err != nil {
		return "", err
	}
	fileDBName := path.Join(screenDir, "filedb.db")
	return fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", fileDBName), nil
}

func MakeFileDB(screenId string) (*sqlx.DB, error) {
	dbUrl, err := MakeFileDBUrl(screenId)
	if err != nil {
		return nil, err
	}
	return sqlx.Open("sqlite3", dbUrl)
}

// will close the DB if not in use (and no waiters)
// returns (closed, closeErr)
// if we cannot close the DB (in use), then we return (false, nil)
// if DB is already closed, we'll return (true, nil)
// if there is an error closing the DB, we'll return (true, err)
// on successful close returns (true, nil)
func (entry *fileDBCacheEntry) CloseDB() (bool, error) {
	entry.CVar.L.Lock()
	defer entry.CVar.L.Unlock()
	if entry.DB == nil {
		return true, nil
	}
	if entry.InUse || entry.Waiters > 0 {
		return false, nil
	}
	err := entry.DB.Close()
	entry.DB = nil
	return true, err
}

// will create DB if doesn't exist
// will Wait() on CVar if InUse
// updates Waiters appropriately
func (entry *fileDBCacheEntry) GetDB() (*sqlx.DB, error) {
	entry.CVar.L.Lock()
	defer entry.CVar.L.Unlock()
	if entry.OpenErr != nil {
		return nil, entry.OpenErr
	}
	entry.Waiters++
	for {
		if entry.InUse {
			entry.CVar.Wait()
			continue
		}
		break
	}
	entry.Waiters--
	if !entry.Migrated {
		FileDBMigrateUp(entry.ScreenId)
		entry.Migrated = true
	}
	if entry.DB == nil {
		db, err := MakeFileDB(entry.ScreenId)
		if err != nil {
			entry.OpenErr = err
			return nil, err
		}
		entry.DB = db
	}
	entry.InUse = true
	entry.LastUse = time.Now()
	return entry.DB, nil
}

func (entry *fileDBCacheEntry) ReleaseDB() {
	entry.CVar.L.Lock()
	defer entry.CVar.L.Unlock()
	entry.InUse = false
	entry.CVar.Signal()
}

func (c *FileDBCache) GetDB(screenId string) (*sqlx.DB, error) {
	entry := c.GetEntry(screenId)
	return entry.GetDB()
}

func (c *FileDBCache) ReleaseDB(screenId string, db *sqlx.DB) {
	entry := c.Cache[screenId]
	entry.ReleaseDB()
}

// fulfills the txwrap DBGetter interface
type FileDBGetter struct {
	ScreenId string
}

func (g FileDBGetter) GetDB(ctx context.Context) (*sqlx.DB, error) {
	return globalFileDBCache.GetDB(g.ScreenId)
}

func (g FileDBGetter) ReleaseDB(db *sqlx.DB) {
	globalFileDBCache.ReleaseDB(g.ScreenId, db)
}

func TryConvertPtyFile(ctx context.Context, screenId string, lineId string) error {
	stat, err := StatCmdPtyFile(ctx, screenId, lineId)
	if err != nil {
		return fmt.Errorf("convert ptyfile, cannot stat: %w", err)
	}
	if stat.DataSize > MaxFileDBInlineFileSize {
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
