package viewbundle

import (
	"errors"

	"github.com/klauspost/compress/zstd"
)

type Zstd struct {
	enc *zstd.Encoder
	dec *zstd.Decoder
}

func NewZstd() (*Zstd, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	return &Zstd{enc: enc, dec: dec}, nil
}

// CompressLens replaces LensBlob with its compressed form and sets LensCodec.
func (z *Zstd) CompressLens(b *Bundle) {
	if b.LensCodec == CodecZstd {
		return
	}
	b.LensBlob = z.enc.EncodeAll(b.LensBlob, make([]byte, 0, len(b.LensBlob)/2))
	b.LensCodec = CodecZstd
}

// DecompressLens returns raw lens bytes; leaves bundle untouched.
// maxOut is a safety cap to prevent decompression bombs.
func (z *Zstd) DecompressLens(b Bundle, maxOut int) ([]byte, error) {
	if b.LensCodec == CodecNone {
		return b.LensBlob, nil
	}
	if b.LensCodec != CodecZstd {
		return nil, errors.New("viewbundle: unsupported codec")
	}
	out, err := z.dec.DecodeAll(b.LensBlob, nil)
	if err != nil {
		return nil, err
	}
	if len(out) > maxOut {
		return nil, errors.New("viewbundle: decompressed lens blob too large")
	}
	return out, nil
}
