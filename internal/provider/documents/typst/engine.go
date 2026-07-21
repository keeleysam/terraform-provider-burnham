// Package typst runs the Typst typesetting engine (compiled to wasm32-wasip1)
// under the pure-Go wazero runtime to render Typst documents to PDF, PNG, SVG,
// and HTML. It is CGO-free. Output is deterministic except for documents that
// call non-deterministic Typst builtins such as datetime.today(), for which the
// module is given a real clock.
package typst

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed typst.wasm
var wasmBytes []byte

// EngineError is a user-caused error from the Typst engine: a document that fails to compile
// (a syntax error, an unknown font, a bad #import). The function layer maps it to an argument
// diagnostic on the source, whereas any other error from Render is an internal fault (a wasm trap,
// a missing result, a decode failure) reported as a general function error.
type EngineError struct{ Msg string }

func (e *EngineError) Error() string { return e.Msg }

var (
	initMu   sync.Mutex
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
)

// ensureCompiled compiles the wasm module once per process; each call instantiates its own module
// (its own memory), so calls don't share state. A failed setup is not cached (see the regex engine
// for the rationale): a transient/cancelled first call must not disable the family for the process.
func ensureCompiled(ctx context.Context) error {
	initMu.Lock()
	defer initMu.Unlock()
	if compiled != nil {
		return nil
	}
	// WithCloseOnContextDone lets wazero interrupt a running compile if Terraform cancels the plan.
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return fmt.Errorf("instantiate WASI: %w", err)
	}
	c, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return fmt.Errorf("compile wasm module: %w", err)
	}
	runtime, compiled = rt, c
	return nil
}

// Request describes one render. Inputs is arbitrary structured data (exposed to the document as
// sys.inputs); Files maps a virtual path to base64-encoded bytes (for #import/#image); Fonts are
// raw font files supplied by the host (bundled + user); PPI is the raster resolution (png only).
type Request struct {
	Op     string
	Source string
	Inputs any
	Files  map[string]string
	Fonts  [][]byte
	PPI    float64
}

// wireRequest is the JSON envelope the wasm shim reads.
type wireRequest struct {
	Op     string            `json:"op"`
	Source string            `json:"source"`
	Inputs any               `json:"inputs,omitempty"`
	Files  map[string]string `json:"files,omitempty"`
	Fonts  []string          `json:"fonts,omitempty"`
	PPI    float64           `json:"ppi,omitempty"`
}

// Render compiles the document and returns its output pages, each as raw bytes: one element for pdf
// and html, one per page for png and svg.
func Render(ctx context.Context, req Request) ([][]byte, error) {
	if err := ensureCompiled(ctx); err != nil {
		return nil, err
	}

	wire := wireRequest{Op: req.Op, Source: req.Source, Inputs: req.Inputs, Files: req.Files, PPI: req.PPI}
	if len(req.Fonts) > 0 {
		wire.Fonts = make([]string, len(req.Fonts))
		for i, f := range req.Fonts {
			wire.Fonts[i] = base64.StdEncoding.EncodeToString(f)
		}
	}
	reqBytes, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBytes, err := callWasm(ctx, reqBytes)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Pages []string `json:"pages"`
		Error string   `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	if resp.Error != "" {
		return nil, &EngineError{Msg: resp.Error}
	}
	if resp.Pages == nil {
		return nil, errors.New("typst engine returned an empty envelope")
	}
	out := make([][]byte, len(resp.Pages))
	for i, p := range resp.Pages {
		b, err := base64.StdEncoding.DecodeString(p)
		if err != nil {
			return nil, fmt.Errorf("decode page %d: %w", i, err)
		}
		out[i] = b
	}
	return out, nil
}

// callWasm instantiates the module, hands it the JSON request, and returns the JSON response bytes.
func callWasm(ctx context.Context, reqBytes []byte) ([]byte, error) {
	// WithSysWalltime gives the module a real clock so datetime.today() reflects the actual date.
	// Documents that use it are non-deterministic by the caller's choice; everything else is stable.
	cfg := wazero.NewModuleConfig().
		WithName("").
		WithStartFunctions("_initialize").
		WithSysWalltime().
		WithStdout(io.Discard).
		WithStderr(io.Discard)
	mod, err := runtime.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}
	defer mod.Close(ctx)

	alloc := mod.ExportedFunction("alloc")
	dealloc := mod.ExportedFunction("dealloc")
	runFn := mod.ExportedFunction("run")
	if alloc == nil || dealloc == nil || runFn == nil {
		return nil, errors.New("wasm module is missing an expected export")
	}
	mem := mod.Memory()

	reqPtr, err := alloc.Call(ctx, uint64(len(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("alloc request: %w", err)
	}
	inPtr := uint32(reqPtr[0])
	if !mem.Write(inPtr, reqBytes) {
		return nil, errors.New("write request out of range")
	}
	defer dealloc.Call(ctx, uint64(inPtr), uint64(len(reqBytes)))

	outLenRes, err := alloc.Call(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("alloc out_len: %w", err)
	}
	outLenPtr := uint32(outLenRes[0])
	defer dealloc.Call(ctx, uint64(outLenPtr), 4)

	res, err := runFn.Call(ctx, uint64(inPtr), uint64(len(reqBytes)), uint64(outLenPtr))
	if err != nil {
		return nil, fmt.Errorf("typst engine trapped: %w", err)
	}
	outPtr := uint32(res[0])

	lenBytes, ok := mem.Read(outLenPtr, 4)
	if !ok {
		return nil, errors.New("read out_len out of range")
	}
	outLen := binary.LittleEndian.Uint32(lenBytes)
	if outPtr == 0 || outLen == 0 {
		return nil, errors.New("typst engine returned no result")
	}
	view, ok := mem.Read(outPtr, outLen)
	if !ok {
		return nil, errors.New("read result out of range")
	}
	result := make([]byte, outLen)
	copy(result, view)
	dealloc.Call(ctx, uint64(outPtr), uint64(outLen))
	return result, nil
}
