package viewbundle

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const wasmURL = "https://raw.githubusercontent.com/shinzonetwork/wasm-bucket/main/bucket/filter_transaction/filter_transaction.wasm"

// We try to load the wasm from ./testdata first (stable, offline-friendly).
// If it's not there, we try to download it from the URL and cache it into ./testdata.
func loadOrFetchWasm(t *testing.T) []byte {
	t.Helper()

	testdataPath := filepath.Join("testdata", "filter_transaction.wasm")

	// 1) Try local file
	if bz, err := os.ReadFile(testdataPath); err == nil && len(bz) > 0 {
		return bz
	}

	// 2) Try fetch and cache
	if err := os.MkdirAll(filepath.Dir(testdataPath), 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(wasmURL)
	if err != nil {
		t.Skipf("could not download wasm (network?): %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Skipf("failed to download wasm: status=%s", resp.Status)
		return nil
	}

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read wasm body: %v", err)
	}
	if len(bz) == 0 {
		t.Fatalf("downloaded wasm is empty")
	}

	// cache best-effort
	_ = os.WriteFile(testdataPath, bz, 0o644)

	return bz
}

func bytesToKB(b int) float64  { return float64(b) / 1024.0 }
func bytesToMB(b int) float64  { return float64(b) / 1_000_000.0 } // decimal MB
func bytesToMiB(b int) float64 { return float64(b) / 1_048_576.0 } // binary MiB

func fmtSize(label string, n int) string {
	return fmt.Sprintf(
		"%-28s %10d bytes | %9.2f KB | %7.2f MB | %7.2f MiB",
		label+":", n, bytesToKB(n), bytesToMB(n), bytesToMiB(n),
	)
}

func Test_View_Bundle_RoundTrip_SizeAndCorrectness(t *testing.T) {
	wasm := loadOrFetchWasm(t)

	// Your example inputs
	query := "Log {address topics data transactionHash blockNumber}"
	sdl := "type Example2 @materialized(if: false) {transactionHash: String}"

	// Build a runtime View where Lens.Path is the blob itself (as base64 string).
	// (This matches what you told me: Path = blob itself.)
	v0 := View{
		Query: query,
		Sdl:   sdl,
		Transform: Transform{
			Lenses: []Lens{
				{
					Path:      base64.StdEncoding.EncodeToString(wasm),
					Arguments: "", // no args for now
				},
			},
		},
	}

	// Measure a "naive JSON view" size (what you currently suffer with).
	naiveJSON, err := json.Marshal(v0)
	if err != nil {
		t.Fatalf("json marshal view: %v", err)
	}

	// Convert View -> Bundle (compress lens region only)
	b0, err := BundleFromView(v0, CodecZstd)
	if err != nil {
		t.Fatalf("BundleFromView: %v", err)
	}

	// Encode bundle to wire bytes (what you send to register(bytes))
	wire0, err := Encode(b0)
	if err != nil {
		t.Fatalf("Encode(bundle): %v", err)
	}

	// Decode wire bytes back to bundle
	b1, err := Decode(wire0)
	if err != nil {
		t.Fatalf("Decode(wire): %v", err)
	}

	// Bundle -> View (decompress lens blob if needed)
	v1, err := ViewFromBundle(b1, 200*1024*1024) // 200MB cap for safety
	if err != nil {
		t.Fatalf("ViewFromBundle: %v", err)
	}

	// ---------------------------
	// Correctness assertions
	// ---------------------------
	if v1.Query != v0.Query {
		t.Fatalf("query mismatch: got=%q want=%q", v1.Query, v0.Query)
	}
	if v1.Sdl != v0.Sdl {
		t.Fatalf("sdl mismatch: got=%q want=%q", v1.Sdl, v0.Sdl)
	}
	if len(v1.Transform.Lenses) != len(v0.Transform.Lenses) {
		t.Fatalf("lens count mismatch: got=%d want=%d", len(v1.Transform.Lenses), len(v0.Transform.Lenses))
	}

	// Arguments should round-trip
	if v1.Transform.Lenses[0].Arguments != v0.Transform.Lenses[0].Arguments {
		t.Fatalf("lens args mismatch: got=%q want=%q", v1.Transform.Lenses[0].Arguments, v0.Transform.Lenses[0].Arguments)
	}

	// Path (base64 wasm) should round-trip exactly
	if v1.Transform.Lenses[0].Path != v0.Transform.Lenses[0].Path {
		t.Fatalf("lens path/base64 mismatch: gotLen=%d wantLen=%d", len(v1.Transform.Lenses[0].Path), len(v0.Transform.Lenses[0].Path))
	}

	// Also verify decoded bytes match original wasm bytes (strong check)
	gotWasm, err := base64.StdEncoding.DecodeString(v1.Transform.Lenses[0].Path)
	if err != nil {
		t.Fatalf("decode returned base64 wasm: %v", err)
	}
	if !bytes.Equal(gotWasm, wasm) {
		t.Fatalf("wasm bytes mismatch after roundtrip")
	}

	// ---------------------------
	// Size reporting (better)
	// ---------------------------
	rawWasmBytes := len(wasm)
	naiveJSONBytes := len(naiveJSON)

	// b0.LensBlob is compressed because codec=zstd
	compressedLensBlobBytes := len(b0.LensBlob)
	wireBytes := len(wire0)

	// Raw lens blob bytes (uncompressed lens region) for lens-only compression %.
	rawLensBlob := EncodeLensBlobOrdered([][]byte{wasm})
	rawLensBlobBytes := len(rawLensBlob)

	// Savings
	savedVsNaive := naiveJSONBytes - wireBytes

	pctVsNaive := 0.0
	if naiveJSONBytes > 0 {
		pctVsNaive = (float64(savedVsNaive) / float64(naiveJSONBytes)) * 100
	}

	pctLensCompression := 0.0
	if rawLensBlobBytes > 0 {
		pctLensCompression = (1.0 - float64(compressedLensBlobBytes)/float64(rawLensBlobBytes)) * 100
	}

	t.Log("---- Size Summary ----")
	t.Log(fmtSize("View JSON (base64 Path)", naiveJSONBytes))
	t.Log(fmtSize("Bundle wire payload", wireBytes))
	t.Log("----------------------")
	t.Log(fmtSize("WASM raw", rawWasmBytes))
	t.Log(fmtSize("Lens blob raw (ordered)", rawLensBlobBytes))
	t.Log(fmtSize("Lens blob stored (zstd)", compressedLensBlobBytes))
	t.Log("----------------------")

	t.Log("---- Savings ----")
	t.Log(fmtSize("Saved vs View JSON", savedVsNaive))
	t.Logf("Saved vs View JSON: %.2f%%", pctVsNaive)

	t.Log(fmtSize("Lens blob saved", rawLensBlobBytes-compressedLensBlobBytes))
	t.Logf("Lens blob compression: %.2f%%", pctLensCompression)
	t.Log("-----------------")

	// Optional note
	if wireBytes >= naiveJSONBytes {
		t.Log("NOTE: wire bytes >= naive JSON here (still correct). Consider tuning zstd or reducing header size.")
	}
}
