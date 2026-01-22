package viewbundle

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const wasmURL = "https://raw.githubusercontent.com/shinzonetwork/wasm-bucket/main/bucket/filter_transaction/filter_transaction.wasm"

func Test_Bundler_FullEncodeDecodeLifecycle(t *testing.T) {
	wasm := mustDownloadWASM(t, wasmURL)

	// Your example inputs
	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"
	args := map[string]any{
		"src":   "address",
		"value": "0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636",
	}
	argsJSON := mustJSON(t, args)

	view := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: string(argsJSON),
				},
			},
		},
	}

	logSizes(t, view, wasm, argsJSON)

	bd := NewBundler()

	wire, err := bd.BundleView(view)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}
	t.Logf("bundled_wire=%s", fmtSize(len(wire)))

	// Sanity: full decode works
	b, err := Decode(wire)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if b.Header.Query != query {
		t.Fatalf("decoded query mismatch: got=%q want=%q", b.Header.Query, query)
	}
	if b.Header.Sdl != sdl {
		t.Fatalf("decoded sdl mismatch: got=%q want=%q", b.Header.Sdl, sdl)
	}

	// Unbundle back to runtime view
	out, err := bd.UnbundleView(wire)
	if err != nil {
		t.Fatalf("UnbundleView: %v", err)
	}

	assertViewEqual(t, view, out)
	assertWasmEqual(t, wasm, out)
}

func Test_Bundler_FullEncode_DecodeHeader_EncodeHeader_Decode(t *testing.T) {
	wasm := mustDownloadWASM(t, wasmURL)

	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"
	args := map[string]any{
		"src":   "address",
		"value": "0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636",
	}
	argsJSON := mustJSON(t, args)

	view := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: string(argsJSON),
				},
			},
		},
	}

	bd := NewBundler()

	wire, err := bd.BundleView(view)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}
	t.Logf("bundled_wire=%s", fmtSize(len(wire)))

	// Header-only decode
	hdr, err := DecodeHeader(wire)
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}

	// Header-only re-encode with NO changes
	wire2, err := EncodeHeader(hdr)
	if err != nil {
		t.Fatalf("EncodeHeader: %v", err)
	}

	// If no header changes, expect byte-for-byte identical payload
	if !bytes.Equal(wire2, wire) {
		t.Fatalf("wire changed after DecodeHeader+EncodeHeader without changes: got=%d want=%d", len(wire2), len(wire))
	}

	// Full decode should work
	b2, err := Decode(wire2)
	if err != nil {
		t.Fatalf("Decode(wire2): %v", err)
	}
	if b2.Header.Query != query {
		t.Fatalf("query mismatch: got=%q want=%q", b2.Header.Query, query)
	}
	if b2.Header.Sdl != sdl {
		t.Fatalf("sdl mismatch: got=%q want=%q", b2.Header.Sdl, sdl)
	}

	// Unbundle should reconstruct same view
	out, err := bd.UnbundleView(wire2)
	if err != nil {
		t.Fatalf("UnbundleView(wire2): %v", err)
	}
	assertViewEqual(t, view, out)
	assertWasmEqual(t, wasm, out)
}

func Test_Bundler_FullEncode_DecodeHeader_UpdateHeader_EncodeHeader_Decode(t *testing.T) {
	wasm := mustDownloadWASM(t, wasmURL)

	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"
	args := map[string]any{
		"src":   "address",
		"value": "0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636",
	}
	argsJSON := mustJSON(t, args)

	view := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: string(argsJSON),
				},
			},
		},
	}

	bd := NewBundler()

	wire, err := bd.BundleView(view)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}

	// Header-only decode
	hdr, err := DecodeHeader(wire)
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}

	// Preserve original lens blob bytes for integrity checks
	origLensBlob := append([]byte(nil), hdr.LensBlob...)

	// Mutate ONLY SDL
	newSDL := hdr.Header.Sdl + " #patched"
	hdr.Header.Sdl = newSDL

	wire2, err := EncodeHeader(hdr)
	if err != nil {
		t.Fatalf("EncodeHeader(after patch): %v", err)
	}
	t.Logf("wire_after_patch=%s", fmtSize(len(wire2)))

	// Full decode and ensure: only SDL changed, blob unchanged
	b2, err := Decode(wire2)
	if err != nil {
		t.Fatalf("Decode(wire2): %v", err)
	}
	if b2.Header.Query != query {
		t.Fatalf("query changed unexpectedly: got=%q want=%q", b2.Header.Query, query)
	}
	if b2.Header.Sdl != newSDL {
		t.Fatalf("sdl not updated: got=%q want=%q", b2.Header.Sdl, newSDL)
	}
	if !bytes.Equal(b2.LensBlob, origLensBlob) {
		t.Fatalf("lens blob changed unexpectedly when only SDL changed")
	}

	// Unbundle and confirm view is the same except SDL
	out, err := bd.UnbundleView(wire2)
	if err != nil {
		t.Fatalf("UnbundleView(wire2): %v", err)
	}
	if out.Query != view.Query {
		t.Fatalf("unbundled query changed unexpectedly: got=%q want=%q", out.Query, view.Query)
	}
	if out.Sdl != newSDL {
		t.Fatalf("unbundled sdl not updated: got=%q want=%q", out.Sdl, newSDL)
	}

	// Args + wasm should still match original
	if len(out.Transform.Lenses) != 1 {
		t.Fatalf("lens count mismatch: got=%d want=1", len(out.Transform.Lenses))
	}
	assertJSONEqual(t, view.Transform.Lenses[0].Arguments, out.Transform.Lenses[0].Arguments)
	assertWasmEqual(t, wasm, out)
}

func Test_Bundler_DecodeHeader_EncodeHeader_PreservesLensBlobAndRefs(t *testing.T) {
	wasm := mustDownloadWASM(t, wasmURL)

	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"
	args := map[string]any{
		"src":   "address",
		"value": "0x1e3aA9fE4Ef01D3cB3189c129a49E3C03126C636",
	}
	argsJSON := mustJSON(t, args)

	view := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: string(argsJSON),
				},
			},
		},
	}

	bd := NewBundler()

	encodedValue, err := bd.BundleView(view)
	if err != nil {
		t.Fatalf("BundleView: %v", err)
	}

	// Decode full for baseline lens fields
	full1, err := Decode(encodedValue)
	if err != nil {
		t.Fatalf("Decode(encodedValue): %v", err)
	}

	// Header-only roundtrip
	hdr, err := DecodeHeader(encodedValue)
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	newEncodedValue, err := EncodeHeader(hdr)
	if err != nil {
		t.Fatalf("EncodeHeader: %v", err)
	}

	full2, err := Decode(newEncodedValue)
	if err != nil {
		t.Fatalf("Decode(newEncodedValue): %v", err)
	}

	// 1) Lens refs preserved (IDs + args)
	if len(full2.Header.Lenses) != len(full1.Header.Lenses) {
		t.Fatalf("lens refs count mismatch: got=%d want=%d", len(full2.Header.Lenses), len(full1.Header.Lenses))
	}
	for i := range full1.Header.Lenses {
		if full2.Header.Lenses[i].ID != full1.Header.Lenses[i].ID {
			t.Fatalf("lens ref ID mismatch at %d: got=%d want=%d", i, full2.Header.Lenses[i].ID, full1.Header.Lenses[i].ID)
		}
		if !bytes.Equal(full2.Header.Lenses[i].Args, full1.Header.Lenses[i].Args) {
			t.Fatalf("lens ref args mismatch at %d", i)
		}
	}

	// 2) Lens blob preserved (this is the actual wasm payload region)
	if full2.LensCodec != full1.LensCodec {
		t.Fatalf("lens codec mismatch: got=%d want=%d", full2.LensCodec, full1.LensCodec)
	}
	if !bytes.Equal(full2.LensBlob, full1.LensBlob) {
		t.Fatalf("lens blob changed after DecodeHeader+EncodeHeader")
	}

	// 3) Optional: strongest possible proof
	if !bytes.Equal(newEncodedValue, encodedValue) {
		t.Fatalf("wire bytes changed after DecodeHeader+EncodeHeader (should be identical)")
	}
}

/* ---------------- assertions ---------------- */

func assertViewEqual(t *testing.T, want, got View) {
	t.Helper()

	if got.Query != want.Query {
		t.Fatalf("Query mismatch: got=%q want=%q", got.Query, want.Query)
	}
	if got.Sdl != want.Sdl {
		t.Fatalf("Sdl mismatch: got=%q want=%q", got.Sdl, want.Sdl)
	}
	if len(got.Transform.Lenses) != len(want.Transform.Lenses) {
		t.Fatalf("Lens count mismatch: got=%d want=%d", len(got.Transform.Lenses), len(want.Transform.Lenses))
	}

	// Arguments: compare as JSON objects to avoid key-order issues
	assertJSONEqual(t, want.Transform.Lenses[0].Arguments, got.Transform.Lenses[0].Arguments)
}

func assertWasmEqual(t *testing.T, wantWasm []byte, gotView View) {
	t.Helper()

	if len(gotView.Transform.Lenses) == 0 {
		t.Fatalf("no lenses in unbundled view")
	}

	gotB64 := gotView.Transform.Lenses[0].Path
	gotWasm, err := base64.StdEncoding.DecodeString(gotB64)
	if err != nil {
		t.Fatalf("decode base64 wasm: %v", err)
	}
	if !bytes.Equal(gotWasm, wantWasm) {
		t.Fatalf("wasm bytes mismatch: got=%d want=%d", len(gotWasm), len(wantWasm))
	}
}

func assertJSONEqual(t *testing.T, a, b string) {
	t.Helper()

	var oa any
	var ob any

	if err := json.Unmarshal([]byte(a), &oa); err != nil {
		t.Fatalf("invalid json in expected args: %v (args=%q)", err, a)
	}
	if err := json.Unmarshal([]byte(b), &ob); err != nil {
		t.Fatalf("invalid json in got args: %v (args=%q)", err, b)
	}

	ja, _ := json.Marshal(oa)
	jb, _ := json.Marshal(ob)

	if !bytes.Equal(ja, jb) {
		t.Fatalf("args json mismatch:\nwant=%s\ngot =%s", string(ja), string(jb))
	}
}

/* ---------------- size reporting ---------------- */

func logSizes(t *testing.T, v View, rawWasm []byte, argsJSON []byte) {
	t.Helper()

	// “view before” as raw pieces (what matters logically)
	beforeRaw := len([]byte(v.Query)) + len([]byte(v.Sdl)) + len(argsJSON) + len(rawWasm)

	// “view runtime” as stored in View (includes base64 wasm string)
	runtimeSize := len([]byte(v.Query)) + len([]byte(v.Sdl)) + len([]byte(v.Transform.Lenses[0].Arguments)) + len([]byte(v.Transform.Lenses[0].Path))

	t.Logf("view_before_raw=%s", fmtSize(beforeRaw))
	t.Logf("view_runtime=%s", fmtSize(runtimeSize))
}

/* ---------------- helpers ---------------- */

func mustDownloadWASM(t *testing.T, url string) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("skipping: cannot download wasm (network?): %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Skipf("skipping: wasm download status=%d", resp.StatusCode)
		return nil
	}

	const max = 20 << 20 // 20 MiB cap
	bz, err := io.ReadAll(io.LimitReader(resp.Body, max))
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}
	if len(bz) == 0 {
		t.Fatalf("downloaded wasm is empty")
	}
	return bz
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	bz, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return bz
}

func fmtSize(n int) string {
	kb := float64(n) / 1024.0
	mb := float64(n) / 1_000_000.0
	return fmt.Sprintf("%d B | %.2f KB | %.2f MB", n, kb, mb)
}
