package viewbundle

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
	"unicode/utf8"
)

func Test_Bundle_StringRoundtrip_InGo_IsLossless(t *testing.T) {
	// Minimal deterministic wasm bytes
	wasm := []byte{0x00, 0x01, 0x02, 0x03, 0x10, 0x20, 0x7f, 0x80, 0x99}

	view := View{
		Query: "Log {address topics data transactionHash blockNumber}",
		Sdl:   "type Example2 @materialized(if: false) {transactionHash: String}",
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: `{"src":"address","value":"0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636"}`,
				},
			},
		},
	}

	bd := NewBundler()

	encoded, err := bd.BundleView(view)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}

	// This is what you're proposing: bytes -> string -> bytes
	roundtrip := []byte(string(encoded))

	// In Go memory, it is lossless.
	if !bytes.Equal(roundtrip, encoded) {
		t.Fatalf("Go string roundtrip changed bytes (unexpected)")
	}

	out, err := bd.UnbundleView(roundtrip)
	if err != nil {
		t.Fatalf("UnbundleView after Go string roundtrip: %v", err)
	}

	if out.Query != view.Query || out.Sdl != view.Sdl {
		t.Fatalf("header mismatch after roundtrip")
	}

	gotWasm, _ := base64.StdEncoding.DecodeString(out.Transform.Lenses[0].Path)
	if !bytes.Equal(gotWasm, wasm) {
		t.Fatalf("wasm mismatch after roundtrip")
	}
}

func Test_Bundle_StringRoundtrip_ThroughJSON_CorruptsPayload(t *testing.T) {
	// Deterministic wasm bytes
	wasm := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x11, 0x22}

	// Force INVALID UTF-8 *inside the header args region* (deterministic).
	// This makes string(encoded) invalid UTF-8, and JSON transport will rewrite it.
	badByte := string([]byte{0xff}) // invalid UTF-8 byte
	args := `{"src":"address","value":"0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636"}` + badByte

	view := View{
		Query: "Log {address topics data transactionHash blockNumber}",
		Sdl:   "type Example2 @materialized(if: false) {transactionHash: String}",
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: args,
				},
			},
		},
	}

	bd := NewBundler()

	encoded, err := bd.BundleView(view)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}

	s := string(encoded)

	// This should be invalid UTF-8 (because we injected 0xff into args bytes).
	if utf8.ValidString(s) {
		// If this ever happens, something changed and the test is no longer proving the point.
		t.Fatalf("expected string(encoded) to be invalid UTF-8, but it was valid")
	}

	// Simulate Cosmos/Tendermint WS / JSON event transport:
	// attribute strings get JSON marshaled/unmarshaled.
	payload := map[string]string{"view": s}

	j, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(j, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	s2 := decoded["view"]
	encoded2 := []byte(s2)

	// It should now be corrupted (JSON replaced invalid bytes with U+FFFD sequences).
	if bytes.Equal(encoded2, encoded) {
		t.Fatalf("expected JSON transport to change bytes, but bytes were identical (unexpected)")
	}

	// And Unbundle/Decode should fail or produce nonsense.
	_, err = bd.UnbundleView(encoded2)
	if err == nil {
		t.Fatalf("expected UnbundleView to fail after JSON transport corruption, but it succeeded")
	}
}

func Test_Bundle_StringEncoded_Unbundle_ViewSemanticsMatch(t *testing.T) {
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00} // tiny wasm header bytes example
	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"
	args := `{"src":"address","value":"0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636"}`

	orig := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{{
				Path:      base64.StdEncoding.EncodeToString(wasm),
				Arguments: args,
			}},
		},
	}

	bd := NewBundler()

	encoded, err := bd.BundleView(orig)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}
	t.Logf("bundle wire size: %s", fmtSize(len(encoded)))

	// The thing you want to test
	encodedString := string(encoded)
	encoded2 := []byte(encodedString)

	out, err := bd.UnbundleView(encoded2)
	if err != nil {
		t.Fatalf("UnbundleView: %v", err)
	}

	logViewSummary(t, "orig", orig)
	logViewSummary(t, "out", out)

	assertViewsEquivalent(t, out, orig)
}

func Test_Bundle_StringEncoded_ThroughJSON_Unbundle_ViewSemantics(t *testing.T) {
	// Use bytes that are likely to produce invalid UTF-8 once bundled.
	// This simulates real bundles that contain arbitrary binary.
	wasm := []byte{
		0x00, 0x61, 0x73, 0x6d, // "\0asm"
		0x01, 0x00, 0x00, 0x00,
		0xff, 0xfe, 0xfd, 0xfc, // invalid UTF-8 bytes inside raw data
		0x10, 0x20, 0x7f, 0x80,
	}

	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"
	args := `{"src":"address","value":"0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636"}`

	orig := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{{
				Path:      base64.StdEncoding.EncodeToString(wasm),
				Arguments: args,
			}},
		},
	}

	bd := NewBundler()

	encoded, err := bd.BundleView(orig)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}

	// Convert to string like Cosmos event attribute would do
	s := string(encoded)

	// This is the key: JSON assumes strings are valid UTF-8.
	// If s contains invalid UTF-8 sequences, JSON marshal/unmarshal will rewrite it.
	t.Logf("string(encoded) utf8Valid=%v", utf8.ValidString(s))

	// Simulate event transport
	payload := map[string]string{"view": s}

	j, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(j, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	s2 := decoded["view"]
	encoded2 := []byte(s2)

	// Log whether bytes changed (we don't "expect identical", but it's useful visibility)
	t.Logf("wireBytes len orig=%d afterJSON=%d", len(encoded), len(encoded2))
	t.Logf("wireBytes equal=%v", bytes.Equal(encoded, encoded2))

	// Now unbundle after JSON transport
	out, err := bd.UnbundleView(encoded2)
	if err == nil {
		// If it somehow succeeds, still check semantics (it may have silently changed)
		// Compare header strings
		if out.Query != orig.Query || out.Sdl != orig.Sdl {
			t.Fatalf("semantic mismatch after JSON transport:\nquery got=%q want=%q\nsdl got=%q want=%q",
				out.Query, orig.Query, out.Sdl, orig.Sdl)
		}
		if len(out.Transform.Lenses) != len(orig.Transform.Lenses) {
			t.Fatalf("lens count mismatch after JSON transport: got=%d want=%d",
				len(out.Transform.Lenses), len(orig.Transform.Lenses))
		}

		// Compare args + wasm bytes
		if out.Transform.Lenses[0].Arguments != orig.Transform.Lenses[0].Arguments {
			t.Fatalf("lens args mismatch after JSON transport:\nGOT:  %q\nWANT: %q",
				out.Transform.Lenses[0].Arguments, orig.Transform.Lenses[0].Arguments)
		}

		gotWasm, err1 := base64.StdEncoding.DecodeString(out.Transform.Lenses[0].Path)
		wantWasm, err2 := base64.StdEncoding.DecodeString(orig.Transform.Lenses[0].Path)
		if err1 != nil || err2 != nil {
			t.Fatalf("base64 decode failed gotErr=%v wantErr=%v", err1, err2)
		}
		if !bytes.Equal(gotWasm, wantWasm) {
			t.Fatalf("wasm bytes mismatch after JSON transport")
		}

		t.Logf("JSON transport did not break this particular payload (rare).")
		return
	}

	// Expected outcome for real-world: corrupted payload, unbundle fails
	t.Logf("UnbundleView failed after JSON transport (expected): %v", err)
}

func mustB64ToBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	return b
}

func logViewSummary(t *testing.T, label string, v View) {
	t.Helper()
	t.Logf("[%s] query_len=%d sdl_len=%d lenses=%d", label, len(v.Query), len(v.Sdl), len(v.Transform.Lenses))
	for i, l := range v.Transform.Lenses {
		t.Logf("[%s] lens[%d] args_len=%d wasm_b64_len=%d", label, i, len(l.Arguments), len(l.Path))
	}
}

func assertViewsEquivalent(t *testing.T, got, want View) {
	t.Helper()

	if got.Query != want.Query {
		t.Fatalf("Query mismatch:\nGOT:  %q\nWANT: %q", got.Query, want.Query)
	}
	if got.Sdl != want.Sdl {
		t.Fatalf("SDL mismatch:\nGOT:  %q\nWANT: %q", got.Sdl, want.Sdl)
	}

	if len(got.Transform.Lenses) != len(want.Transform.Lenses) {
		t.Fatalf("lens count mismatch: got=%d want=%d", len(got.Transform.Lenses), len(want.Transform.Lenses))
	}

	for i := range want.Transform.Lenses {
		if got.Transform.Lenses[i].Arguments != want.Transform.Lenses[i].Arguments {
			t.Fatalf("lens[%d] args mismatch:\nGOT:  %q\nWANT: %q", i, got.Transform.Lenses[i].Arguments, want.Transform.Lenses[i].Arguments)
		}

		// Either compare base64 strings directly...
		if got.Transform.Lenses[i].Path != want.Transform.Lenses[i].Path {
			// ...or do a stronger check by comparing decoded bytes
			gb := mustB64ToBytes(t, got.Transform.Lenses[i].Path)
			wb := mustB64ToBytes(t, want.Transform.Lenses[i].Path)
			if !bytes.Equal(gb, wb) {
				t.Fatalf("lens[%d] wasm bytes mismatch (base64 differs and bytes differ)", i)
			}
		}
	}
}
