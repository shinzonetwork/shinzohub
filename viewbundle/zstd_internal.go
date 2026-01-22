package viewbundle

import (
	"sync"

	"github.com/klauspost/compress/zstd"
)

var (
	zOnce sync.Once
	zEnc  *zstd.Encoder
	zDec  *zstd.Decoder
	zErr  error
)

func initZ() {
	zDec, zErr = zstd.NewReader(nil)
	if zErr != nil {
		return
	}
	zEnc, zErr = zstd.NewWriter(nil)
}

func compressRawLensBlob(raw []byte) ([]byte, error) {
	zOnce.Do(initZ)
	if zErr != nil {
		return nil, zErr
	}
	return zEnc.EncodeAll(raw, make([]byte, 0, len(raw)/2)), nil
}

func decompressRawLensBlob(comp []byte, maxOut int) ([]byte, error) {
	zOnce.Do(initZ)
	if zErr != nil {
		return nil, zErr
	}
	out, err := zDec.DecodeAll(comp, nil)
	if err != nil {
		return nil, err
	}
	if maxOut > 0 && len(out) > maxOut {
		return nil, ErrDecompressedTooLarge
	}
	return out, nil
}
