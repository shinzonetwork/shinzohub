package viewbundle

import (
	"errors"
	"sync"

	"github.com/klauspost/compress/zstd"
)

var (
	zstdOnce sync.Once
	zEnc     *zstd.Encoder
	zDec     *zstd.Decoder
	zErr     error
)

func initZstd() {
	zDec, zErr = zstd.NewReader(nil)
	if zErr != nil {
		return
	}
	zEnc, zErr = zstd.NewWriter(nil) // tune options later if needed
}

func CompressLensBlob(raw []byte) ([]byte, error) {
	zstdOnce.Do(initZstd)
	if zErr != nil {
		return nil, zErr
	}
	if zEnc == nil {
		return nil, errors.New("viewbundle: zstd encoder not initialized")
	}
	return zEnc.EncodeAll(raw, make([]byte, 0, len(raw)/2)), nil
}

func DecompressLensBlob(comp []byte, maxOut int) ([]byte, error) {
	zstdOnce.Do(initZstd)
	if zErr != nil {
		return nil, zErr
	}
	if zDec == nil {
		return nil, errors.New("viewbundle: zstd decoder not initialized")
	}
	out, err := zDec.DecodeAll(comp, nil)
	if err != nil {
		return nil, err
	}
	if maxOut > 0 && len(out) > maxOut {
		return nil, ErrTooLarge
	}
	return out, nil
}
