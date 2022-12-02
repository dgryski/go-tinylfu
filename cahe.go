package main

import (
	"encoding/binary"
	"unsafe"
)

type NewCacheFunc func(size int) Cache

type Cache interface {
	Name() string
	Set(string)
	Get(string) bool
	Close()
}

func stringFromInt64(n int64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(n))
	return bytesToString(b)
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
