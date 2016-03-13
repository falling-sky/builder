package fileutil

import (
	"io/ioutil"
	"sync"
)

type ReadFileCacheItem struct {
	s string
	e error
}
type ReadFileCacheType struct {
	lock   sync.RWMutex
	byname map[string]ReadFileCacheItem
}

var ReadFileCache ReadFileCacheType

func init() {
	ReadFileCache.byname = make(map[string]ReadFileCacheItem)
}

func ReadFileFromDisk(fn string) (string, error) {
	b, e := ioutil.ReadFile(fn)
	return string(b), e
}

func ReadFileWithCache(fn string) (string, error) {
	ReadFileCache.lock.Lock()
	defer ReadFileCache.lock.Unlock()
	if item, ok := ReadFileCache.byname[fn]; ok {
		return item.s, item.e
	}

	// Crap. Go read it for real.
	s, e := ReadFileFromDisk(fn)
	ReadFileCache.byname[fn] = ReadFileCacheItem{s: s, e: e}
	return s, e
}
