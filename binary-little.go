package main

import (
	"fmt"
)

func read_u16(buf []byte, off int, value *uint16) (int, error) {
	if off+1 >= len(buf) {
		return 0, fmt.Errorf("Out of range")
	}
	*value = uint16(buf[off]) << 8
	*value |= uint16(buf[off+1])
	return off + 1, nil
}

func write_u16(buf []byte, off int, value uint16) (int, error) {
	if off+1 >= len(buf) {
		return 0, fmt.Errorf("Out of range")
	}
	buf[off] = uint8(value >> 8)
	buf[off+1] = uint8(value)
	return off + 2, nil
}

func write_u32(buf []byte, off int, value uint32) (int, error) {
	if off+3 >= len(buf) {
		return 0, fmt.Errorf("Out of range")
	}
	buf[off] = uint8(value >> 24)
	buf[off+1] = uint8(value >> 16)
	buf[off+2] = uint8(value >> 8)
	buf[off+3] = uint8(value)
	return off + 4, nil
}
