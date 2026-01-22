package viewbundle

const (
	Magic   = "VWL"
	Version = 1

	CodecNone = 0
	CodecZstd = 1
)

type View struct {
	Query     string
	Sdl       string
	Transform Transform
}

type Transform struct {
	Lenses []Lens
}

type Lens struct {
	Path      string // base64(WASM bytes)
	Arguments string
}

type Bundle struct {
	Header    Header
	LensCodec uint8
	LensBlob  []byte // ordered wasm bytes (raw or compressed)
}

type Header struct {
	Query  string
	Sdl    string
	Lenses []LensRef // ordered; IDs are positional (1..N)
}

type LensRef struct {
	ID   uint32 // 1..N by position
	Args []byte
}
