package viewbundle

import (
	"bytes"
	"fmt"
)

// DecodedHeader is a “header-only” decode result that still carries the lens blob bytes.
// LensBlob is a slice pointing into the original wire (no copy).
type DecodedHeader struct {
	Header    Header
	LensCodec uint8
	LensBlob  []byte // references original wire bytes (raw or compressed)
}

// DecodeHeader parses only the header + codec + lens blob length,
// but does NOT copy the lens blob bytes (it slices into wire).
func DecodeHeader(wire []byte) (DecodedHeader, error) {
	return decodeHeaderWithLimits(wire, DefaultLimits())
}

func decodeHeaderWithLimits(wire []byte, lim Limits) (DecodedHeader, error) {
	if lim.MaxWireBytes > 0 && len(wire) > lim.MaxWireBytes {
		return DecodedHeader{}, tooLarge("wire", len(wire), lim.MaxWireBytes)
	}

	if len(wire) < 3+1+4+4+2+1+4 {
		return DecodedHeader{}, ErrCorrupt
	}

	if string(wire[:3]) != Magic {
		return DecodedHeader{}, ErrBadMagic
	}

	if int(wire[3]) != Version {
		return DecodedHeader{}, ErrBadVersion
	}

	i := 4

	// Query
	qLen, ni, ok := readU32(wire, i)
	if !ok {
		return DecodedHeader{}, ErrCorrupt
	}

	i = ni
	if int(qLen) > lim.MaxQueryBytes || i+int(qLen) > len(wire) {
		if int(qLen) > lim.MaxQueryBytes {
			return DecodedHeader{}, tooLarge("header.query", int(qLen), lim.MaxQueryBytes)
		}
		return DecodedHeader{}, ErrCorrupt
	}

	query := string(wire[i : i+int(qLen)])
	i += int(qLen)

	// SDL
	sLen, ni, ok := readU32(wire, i)
	if !ok {
		return DecodedHeader{}, ErrCorrupt
	}

	i = ni
	if int(sLen) > lim.MaxSdlBytes || i+int(sLen) > len(wire) {
		if int(sLen) > lim.MaxSdlBytes {
			return DecodedHeader{}, tooLarge("header.sdl", int(sLen), lim.MaxSdlBytes)
		}
		return DecodedHeader{}, ErrCorrupt
	}

	sdl := string(wire[i : i+int(sLen)])
	i += int(sLen)

	// Lens ref count
	lc, ni, ok := readU16(wire, i)
	if !ok {
		return DecodedHeader{}, ErrCorrupt
	}

	i = ni
	if int(lc) > lim.MaxLensRefs {
		return DecodedHeader{}, tooLarge("header.lenses.count", int(lc), lim.MaxLensRefs)
	}

	lensRefs := make([]LensRef, 0, lc)
	for j := 0; j < int(lc); j++ {
		id, ni, ok := readU32(wire, i)
		if !ok {
			return DecodedHeader{}, ErrCorrupt
		}
		i = ni

		aLen, ni, ok := readU32(wire, i)
		if !ok {
			return DecodedHeader{}, ErrCorrupt
		}
		i = ni

		if int(aLen) > lim.MaxArgsBytes || i+int(aLen) > len(wire) {
			if int(aLen) > lim.MaxArgsBytes {
				return DecodedHeader{}, tooLarge(fmt.Sprintf("header.lenses[%d].args", j+1), int(aLen), lim.MaxArgsBytes)
			}
			return DecodedHeader{}, ErrCorrupt
		}

		args := wire[i : i+int(aLen)] //
		i += int(aLen)

		lensRefs = append(lensRefs, LensRef{ID: id, Args: args})
	}

	// enforce positional IDs
	for idx := range lensRefs {
		if lensRefs[idx].ID != uint32(idx+1) {
			return DecodedHeader{}, ErrCorrupt
		}
	}

	// codec
	if i >= len(wire) {
		return DecodedHeader{}, ErrCorrupt
	}

	codec := wire[i]

	i++
	if codec != CodecNone && codec != CodecZstd {
		return DecodedHeader{}, ErrCodec
	}

	// lens blob len
	lbLen, ni, ok := readU32(wire, i)
	if !ok {
		return DecodedHeader{}, ErrCorrupt
	}
	i = ni

	if int(lbLen) > lim.MaxLensBlobBytes || i+int(lbLen) > len(wire) {
		if int(lbLen) > lim.MaxLensBlobBytes {
			return DecodedHeader{}, tooLarge("lensBlob", int(lbLen), lim.MaxLensBlobBytes)
		}
		return DecodedHeader{}, ErrCorrupt
	}

	lensBlob := wire[i : i+int(lbLen)]
	i += int(lbLen)

	if i != len(wire) {
		return DecodedHeader{}, ErrCorrupt
	}

	return DecodedHeader{
		Header: Header{
			Query:  query,
			Sdl:    sdl,
			Lenses: lensRefs,
		},
		LensCodec: codec,
		LensBlob:  lensBlob,
	}, nil
}

// EncodeHeader re-encodes a DecodedHeader back into wire bytes.
// It does NOT compress/decompress; it just reuses LensCodec + LensBlob as-is.
func EncodeHeader(h DecodedHeader) ([]byte, error) {
	return encodeHeaderWithLimits(h, DefaultLimits())
}

func encodeHeaderWithLimits(h DecodedHeader, lim Limits) ([]byte, error) {
	if len(h.Header.Query) > lim.MaxQueryBytes {
		return nil, tooLarge("header.query", len(h.Header.Query), lim.MaxQueryBytes)
	}

	if len(h.Header.Sdl) > lim.MaxSdlBytes {
		return nil, tooLarge("header.sdl", len(h.Header.Sdl), lim.MaxSdlBytes)
	}

	if len(h.Header.Lenses) > lim.MaxLensRefs {
		return nil, tooLarge("header.lenses.count", len(h.Header.Lenses), lim.MaxLensRefs)
	}

	for i, lr := range h.Header.Lenses {
		if lr.ID != uint32(i+1) {
			return nil, ErrCorrupt
		}

		if len(lr.Args) > lim.MaxArgsBytes {
			return nil, tooLarge(fmt.Sprintf("header.lenses[%d].args", i+1), len(lr.Args), lim.MaxArgsBytes)
		}
	}

	if len(h.LensBlob) > lim.MaxLensBlobBytes {
		return nil, tooLarge("lensBlob", len(h.LensBlob), lim.MaxLensBlobBytes)
	}

	if h.LensCodec != CodecNone && h.LensCodec != CodecZstd {
		return nil, ErrCodec
	}

	// capacity hint
	capHint := 3 + 1 +
		4 + len(h.Header.Query) +
		4 + len(h.Header.Sdl) +
		2

	for _, lr := range h.Header.Lenses {
		capHint += 4 + 4 + len(lr.Args)
	}

	capHint += 1 + 4 + len(h.LensBlob)

	if lim.MaxWireBytes > 0 && capHint > lim.MaxWireBytes {
		return nil, tooLarge("wire", capHint, lim.MaxWireBytes)
	}

	var buf bytes.Buffer
	buf.Grow(capHint)

	buf.WriteString(Magic)
	buf.WriteByte(byte(Version))

	// Query
	if err := writeU32(&buf, uint32(len(h.Header.Query))); err != nil {
		return nil, err
	}

	buf.WriteString(h.Header.Query)

	// SDL
	if err := writeU32(&buf, uint32(len(h.Header.Sdl))); err != nil {
		return nil, err
	}

	buf.WriteString(h.Header.Sdl)

	// Lens refs
	if err := writeU16(&buf, uint16(len(h.Header.Lenses))); err != nil {
		return nil, err
	}

	for _, lr := range h.Header.Lenses {
		if err := writeU32(&buf, lr.ID); err != nil {
			return nil, err
		}
		if err := writeU32(&buf, uint32(len(lr.Args))); err != nil {
			return nil, err
		}
		buf.Write(lr.Args)
	}

	// codec + lens blob
	buf.WriteByte(h.LensCodec)
	if err := writeU32(&buf, uint32(len(h.LensBlob))); err != nil {
		return nil, err
	}

	buf.Write(h.LensBlob)

	out := buf.Bytes()
	if lim.MaxWireBytes > 0 && len(out) > lim.MaxWireBytes {
		return nil, tooLarge("wire", len(out), lim.MaxWireBytes)
	}

	return out, nil
}
