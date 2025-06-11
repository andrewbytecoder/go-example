package linux

import (
	"encoding/binary"
	"unsafe"
)

// NativeEndian is the ByteOrder of the current system.
var NativeEndian binary.ByteOrder

// 借助init函数在函数启动过程中实现对系统字节序的判断(大小端)

func init() {
	// Examine the memory layout of an int16 to determine system
	// endianness.
	var one int16 = 1
	b := (*byte)(unsafe.Pointer(&one))
	if *b == 0 {
		NativeEndian = binary.BigEndian
	} else {
		NativeEndian = binary.LittleEndian
	}
}

func NativelyLittle() bool {
	return NativeEndian == binary.LittleEndian
}
