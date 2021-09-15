package tcpclient

import (
	"encoding/binary"
	"unsafe"
)

func GetHeader(command byte) []byte {
	return []byte{command}
}

func Uint32ToBytes(a uint32) []byte {
	size := unsafe.Sizeof(a)
	bs := make([]byte, size)
	binary.BigEndian.PutUint32(bs, a)
	return bs
}
