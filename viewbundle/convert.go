package viewbundle

import (
	"encoding/base64"
	"fmt"
)

// BundleFromView converts a runtime View into a Bundle.
// Assumes Lens.Path is base64-encoded WASM bytes.
// Lens IDs are positional: 1..N.
func BundleFromView(v View, codec uint8) (Bundle, error) {
	refs := make([]LensRef, 0, len(v.Transform.Lenses))
	wasms := make([][]byte, 0, len(v.Transform.Lenses))

	for idx, l := range v.Transform.Lenses {
		id := uint32(idx + 1)

		wasm, err := base64.StdEncoding.DecodeString(l.Path)
		if err != nil {
			return Bundle{}, fmt.Errorf("viewbundle: lens %d Path must be base64 wasm: %w", id, err)
		}

		refs = append(refs, LensRef{
			ID:   id,
			Args: []byte(l.Arguments),
		})
		wasms = append(wasms, wasm)
	}

	rawLensBlob := EncodeLensBlobOrdered(wasms)

	out := Bundle{
		Header: Header{
			Query:  v.Query,
			Sdl:    v.Sdl,
			Lenses: refs,
		},
		LensCodec: codec,
		LensBlob:  rawLensBlob,
	}

	switch codec {
	case CodecNone:
		return out, nil
	case CodecZstd:
		comp, err := CompressLensBlob(rawLensBlob)
		if err != nil {
			return Bundle{}, err
		}
		out.LensBlob = comp
		return out, nil
	default:
		return Bundle{}, ErrCodec
	}
}

// ViewFromBundle converts a Bundle back into a runtime View.
// It will base64-encode each lens wasm into Lens.Path.
func ViewFromBundle(b Bundle, maxDecompressed int) (View, error) {
	var rawLensBlob []byte

	switch b.LensCodec {
	case CodecNone:
		rawLensBlob = b.LensBlob
	case CodecZstd:
		out, err := DecompressLensBlob(b.LensBlob, maxDecompressed)
		if err != nil {
			return View{}, err
		}
		rawLensBlob = out
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
