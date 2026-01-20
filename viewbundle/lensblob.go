package viewbundle

import (
	"bytes"
	"encoding/binary"
)

// Option 1: ordered wasm list.
// LensRef order in Header.Lenses MUST match the wasm order in this blob.
//
// Layout:
//
//	COUNT u16
//	repeat COUNT:
//	  WASM_LEN u32
//	  WASM_BYTES
func EncodeLensBlobOrdered(wasms [][]byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(wasms)))
	for _, w := range wasms {
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(w)))
		buf.Write(w)
	}
	return buf.Bytes()
}

func DecodeLensBlobOrdered(bz []byte) ([][]byte, error) {
	if len(bz) < 2 {
		return nil, ErrCorrupt
	}
	i := 0
	n := int(binary.LittleEndian.Uint16(bz[i : i+2]))
	i += 2

	out := make([][]byte, 0, n)
	for j := 0; j < n; j++ {
		if i+4 > len(bz) {
			return nil, ErrCorrupt
		}
		l := int(binary.LittleEndian.Uint32(bz[i : i+4]))
		i += 4
		if l < 0 || i+l > len(bz) {
			return nil, ErrCorrupt
		}

		wasm := make([]byte, l)
		copy(wasm, bz[i:i+l])
		i += l
		out = append(out, wasm)
	}
	return out, nil
}
