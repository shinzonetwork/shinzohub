package viewbundle

type Limits struct {
	MaxWireBytes     int
	MaxQueryBytes    int
	MaxSdlBytes      int
	MaxArgsBytes     int
	MaxLensRefs      int
	MaxLensBlobBytes int
}

func DefaultLimits() Limits {
	return Limits{
		MaxWireBytes:     384 * 1024,       // 384 KiB wire => base64 <= 512 KiB
		MaxQueryBytes:    48 * 1024,        // 48 KiB (query should not be huge)
		MaxSdlBytes:      64 * 1024,        // 64 KiB (SDL can be a bit bigger)
		MaxArgsBytes:     128 * 1024,       // 128 KiB per lens args (ABI JSON fits)
		MaxLensRefs:      128,              // keep header overhead sane
		MaxLensBlobBytes: 10 * 1024 * 1024, // safety cap; wire cap is the real limiter
	}
}
