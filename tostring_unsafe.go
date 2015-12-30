// +build !appengine

package tinylfu

import (
	"reflect"
	"unsafe"
)

func stringToSlice(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := &reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	bp := (*[]byte)(unsafe.Pointer(bh))
	return *bp
}
