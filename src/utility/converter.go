package utility

import (
	"unsafe"
	"reflect"
)
func BytesToInt32(bytes []byte)  int32{
	return *(*int32)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&bytes)).Data))
}

func GetBytesFromInt32(v int32, buf []byte)  {
	ptr := uintptr(unsafe.Pointer(&v))
	s := &reflect.SliceHeader{Data:ptr, Len:4, Cap:4}
	b := *(*[]byte)(unsafe.Pointer(s))
	copy(buf, b)
}