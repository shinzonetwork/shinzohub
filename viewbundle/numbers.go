package viewbundle

import (
	"bytes"
	"encoding/binary"
)

func writeU16(buf *bytes.Buffer, v uint16) error {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	_, err := buf.Write(b[:])
	return err
}

func writeU32(buf *bytes.Buffer, v uint32) error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	_, err := buf.Write(b[:])
	return err
}

func readU16(bz []byte, i int) (uint16, int, bool) {
	if i+2 > len(bz) {
		return 0, i, false
	}
	return binary.LittleEndian.Uint16(bz[i : i+2]), i + 2, true
}

func readU32(bz []byte, i int) (uint32, int, bool) {
	if i+4 > len(bz) {
		return 0, i, false
	}
	return binary.LittleEndian.Uint32(bz[i : i+4]), i + 4, true
}
