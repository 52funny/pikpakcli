package utils

import (
	"reflect"
	"unsafe"
)

func StringToByteSlice(str string) (bs []byte) {
	strH := (*reflect.StringHeader)(unsafe.Pointer(&str))
	bsH := (*reflect.SliceHeader)(unsafe.Pointer(&bs))
	bsH.Data = strH.Data
	bsH.Len = strH.Len
	bsH.Cap = strH.Len
	return
}

func ByteSliceToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
