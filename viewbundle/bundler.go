package viewbundle

import (
	"encoding/base64"
	"fmt"
)

var ErrDecompressedTooLarge = fmt.Errorf("viewbundle: decompressed lens blob too large")

// Bundler is the single entrypoint for the “bundle/unbundle view” concept.
// No codec selection. Always uses zstd for LensBlob.
type Bundler struct {
	// Safety cap for decompression during UnbundleView.
	// If 0, defaults to DefaultLimits().MaxLensBlobBytes.
	MaxDecompressedLensBlob int
}

func NewBundler() Bundler {
	return Bundler{MaxDecompressedLensBlob: 0}
}

// BundleView converts a runtime View into the wire Bundle bytes (register payload).
// Internally it:
// - base64-decodes lens wasm from Lens.Path
// - builds header refs (IDs 1..N)
// - packs ordered wasm list into raw lens blob
// - compresses raw lens blob with zstd
// - encodes to wire bytes with DefaultLimits enforced
func (bd Bundler) BundleView(v View) ([]byte, error) {
	refs := make([]LensRef, 0, len(v.Transform.Lenses))
	wasms := make([][]byte, 0, len(v.Transform.Lenses))

	for idx, l := range v.Transform.Lenses {
		id := uint32(idx + 1)

		wasm, err := base64.StdEncoding.DecodeString(l.Path)
		if err != nil {
			return nil, fmt.Errorf("viewbundle: lens %d Path must be base64 wasm: %w", id, err)
		}

		refs = append(refs, LensRef{
			ID:   id,
			Args: []byte(l.Arguments),
		})
		wasms = append(wasms, wasm)
	}

	rawLensBlob := EncodeLensBlobOrdered(wasms)

	comp, err := compressRawLensBlob(rawLensBlob)
	if err != nil {
		return nil, fmt.Errorf("viewbundle: compress lens blob: %w", err)
	}

	b := Bundle{
		Header: Header{
			Query:  v.Query,
			Sdl:    v.Sdl,
			Lenses: refs,
		},
		LensCodec: CodecZstd,
		LensBlob:  comp,
	}

	wire, err := Encode(b)
	if err != nil {
		return nil, err
	}
	return wire, nil
}

// UnbundleView converts wire bytes back into a runtime View (lens wasm base64 in Lens.Path).
// It:
// - decodes wire bytes
// - decompresses lens blob safely (cap)
// - splits ordered wasm list
// - rebuilds View + lenses
func (bd Bundler) UnbundleView(wire []byte) (View, error) {
	b, err := Decode(wire)
	if err != nil {
		return View{}, err
	}

	maxOut := bd.MaxDecompressedLensBlob
	if maxOut <= 0 {
		maxOut = DefaultLimits().MaxLensBlobBytes
	}

	var rawLensBlob []byte
	switch b.LensCodec {
	case CodecZstd:
		out, err := decompressRawLensBlob(b.LensBlob, maxOut)
		if err != nil {
			return View{}, err
		}
		rawLensBlob = out
	case CodecNone:
		// (We don't produce this, but allow decoding older bundles.)
		rawLensBlob = b.LensBlob
	default:
		return View{}, ErrCodec
	}

	wasms, err := DecodeLensBlobOrdered(rawLensBlob)
	if err != nil {
		return View{}, err
	}
	if len(wasms) != len(b.Header.Lenses) {
		return View{}, ErrMismatch
	}

	lenses := make([]Lens, 0, len(wasms))
	for i := range wasms {
		lenses = append(lenses, Lens{
			Path:      base64.StdEncoding.EncodeToString(wasms[i]),
			Arguments: string(b.Header.Lenses[i].Args),
		})
	}

	return View{
		Query: b.Header.Query,
		Sdl:   b.Header.Sdl,
		Transform: Transform{
			Lenses: lenses,
		},
	}, nil
}
