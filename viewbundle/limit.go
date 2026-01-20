package viewbundle

type Limits struct {
	MaxQueryBytes    int
	MaxSdlBytes      int
	MaxArgsBytes     int
	MaxLensRefs      int
	MaxLensBlobBytes int
}

func DefaultLimits() Limits {
	return Limits{
		MaxQueryBytes:    512 * 1024,       // 512 KB
		MaxSdlBytes:      512 * 1024,       // 512 KB
		MaxArgsBytes:     512 * 1024,       // 512 KB
		MaxLensRefs:      2048,             // up to 2,048 lenses
		MaxLensBlobBytes: 50 * 1024 * 1024, // 50 MB (raw or compressed bytes as carried)
	}
}
