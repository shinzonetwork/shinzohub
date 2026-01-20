package viewbundle

import (
	"bytes"
)

func Encode(b Bundle) ([]byte, error) {
	return encodeWithLimits(b, DefaultLimits())
}

func Decode(bz []byte) (Bundle, error) {
	return decodeWithLimits(bz, DefaultLimits())
}

func encodeWithLimits(b Bundle, lim Limits) ([]byte, error) {
	// Validate sizes
	if len(b.Header.Query) > lim.MaxQueryBytes || len(b.Header.Sdl) > lim.MaxSdlBytes {
		return nil, ErrTooLarge
	}
	if len(b.Header.Lenses) > lim.MaxLensRefs {
		return nil, ErrTooLarge
	}
	for _, lr := range b.Header.Lenses {
		if len(lr.Args) > lim.MaxArgsBytes {
			return nil, ErrTooLarge
		}
	}
	if len(b.LensBlob) > lim.MaxLensBlobBytes {
		return nil, ErrTooLarge
	}
	if b.LensCodec != CodecNone && b.LensCodec != CodecZstd {
		return nil, ErrCodec
	}

	// capacity hint
	capHint := 3 + 1 +
		4 + len(b.Header.Query) +
		4 + len(b.Header.Sdl) +
		2
	for _, lr := range b.Header.Lenses {
		capHint += 4 + 4 + len(lr.Args)
	}
	capHint += 1 + 4 + len(b.LensBlob)

	var buf bytes.Buffer
	buf.Grow(capHint)

	buf.WriteString(Magic)
	buf.WriteByte(byte(Version))

	// Query
	if err := writeU32(&buf, uint32(len(b.Header.Query))); err != nil {
		return nil, err
	}
	buf.WriteString(b.Header.Query)

	// SDL
	if err := writeU32(&buf, uint32(len(b.Header.Sdl))); err != nil {
		return nil, err
	}
	buf.WriteString(b.Header.Sdl)

	// Lens refs (ordered)
	if err := writeU16(&buf, uint16(len(b.Header.Lenses))); err != nil {
		return nil, err
	}
	for idx, lr := range b.Header.Lenses {
		// enforce positional ids 1..N
		if lr.ID != uint32(idx+1) {
			return nil, ErrCorrupt
		}
		if err := writeU32(&buf, lr.ID); err != nil {
			return nil, err
		}
		if err := writeU32(&buf, uint32(len(lr.Args))); err != nil {
			return nil, err
		}
		buf.Write(lr.Args)
	}

	// Lens blob
	buf.WriteByte(b.LensCodec)
	if err := writeU32(&buf, uint32(len(b.LensBlob))); err != nil {
		return nil, err
	}
	buf.Write(b.LensBlob)

	return buf.Bytes(), nil
}

func decodeWithLimits(bz []byte, lim Limits) (Bundle, error) {
	if len(bz) < 3+1+4+4+2+1+4 {
		return Bundle{}, ErrCorrupt
	}
	if string(bz[:3]) != Magic {
		return Bundle{}, ErrBadMagic
	}
	if int(bz[3]) != Version {
		return Bundle{}, ErrBadVersion
	}

	i := 4

	// Query
	qLen, ni, ok := readU32(bz, i)
	if !ok {
		return Bundle{}, ErrCorrupt
	}
	i = ni
	if int(qLen) > lim.MaxQueryBytes || i+int(qLen) > len(bz) {
		return Bundle{}, ErrTooLarge
	}
	query := string(bz[i : i+int(qLen)])
	i += int(qLen)

	// SDL
	sLen, ni, ok := readU32(bz, i)
	if !ok {
		return Bundle{}, ErrCorrupt
	}
	i = ni
	if int(sLen) > lim.MaxSdlBytes || i+int(sLen) > len(bz) {
		return Bundle{}, ErrTooLarge
	}
	sdl := string(bz[i : i+int(sLen)])
	i += int(sLen)

	// Lens ref count
	lc, ni, ok := readU16(bz, i)
	if !ok {
		return Bundle{}, ErrCorrupt
	}
	i = ni
	if int(lc) > lim.MaxLensRefs {
		return Bundle{}, ErrTooLarge
	}

	lensRefs := make([]LensRef, 0, lc)
	for j := 0; j < int(lc); j++ {
		id, ni, ok := readU32(bz, i)
		if !ok {
			return Bundle{}, ErrCorrupt
		}
		i = ni

		aLen, ni, ok := readU32(bz, i)
		if !ok {
			return Bundle{}, ErrCorrupt
		}
		i = ni

		if int(aLen) > lim.MaxArgsBytes || i+int(aLen) > len(bz) {
			return Bundle{}, ErrTooLarge
		}

		args := make([]byte, int(aLen))
		copy(args, bz[i:i+int(aLen)])
		i += int(aLen)

		lensRefs = append(lensRefs, LensRef{ID: id, Args: args})
	}

	// codec
	if i >= len(bz) {
		return Bundle{}, ErrCorrupt
	}
	codec := bz[i]
	i++
	if codec != CodecNone && codec != CodecZstd {
		return Bundle{}, ErrCodec
	}

	// lens blob
	lbLen, ni, ok := readU32(bz, i)
	if !ok {
		return Bundle{}, ErrCorrupt
	}
	i = ni
	if int(lbLen) > lim.MaxLensBlobBytes || i+int(lbLen) > len(bz) {
		return Bundle{}, ErrTooLarge
	}
	lensBlob := make([]byte, int(lbLen))
	copy(lensBlob, bz[i:i+int(lbLen)])

	// enforce positional IDs if present
	for idx := range lensRefs {
		if lensRefs[idx].ID != uint32(idx+1) {
			return Bundle{}, ErrCorrupt
		}
	}

	return Bundle{
		Header: Header{
			Query:  query,
			Sdl:    sdl,
			Lenses: lensRefs,
		},
		LensCodec: codec,
		LensBlob:  lensBlob,
	}, nil
}
