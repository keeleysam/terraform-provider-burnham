// Package resvg renders SVG to PNG by running resvg (compiled to wasm32-wasip1)
// under the pure-Go wazero runtime. It is CGO-free and deterministic: the same
// SVG + fonts produce byte-identical output across host architectures.
//
// The wasm module is built from the Rust shim under rust/ (see gen.go) and is
// gitignored, so a full build must run `go generate ./...` first to produce it.
package resvg

import (
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed svg_render.wasm
var wasmBytes []byte

var (
	initOnce sync.Once
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	initErr  error
)

// ensureCompiled compiles the wasm module once for the process. The compiled
// module is reused across renders; each render instantiates its own module (its
// own linear memory), so concurrent renders do not share state.
func ensureCompiled(ctx context.Context) error {
	initOnce.Do(func() {
		rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
			initErr = fmt.Errorf("instantiate WASI: %w", err)
			rt.Close(ctx)
			return
		}
		c, err := rt.CompileModule(ctx, wasmBytes)
		if err != nil {
			initErr = fmt.Errorf("compile wasm module: %w", err)
			rt.Close(ctx)
			return
		}
		runtime, compiled = rt, c
	})
	return initErr
}

// Render rasterizes svg to PNG bytes. fonts are raw TTF/OTF font files loaded
// into resvg's in-memory fontdb. If width and height are both > 0 the SVG is
// scaled to that exact pixel box; if only one is > 0 the other is derived from
// the SVG's aspect ratio; if neither is set the intrinsic size times scale is
// used (scale <= 0 means 1.0).
func Render(ctx context.Context, svg []byte, fonts [][]byte, width, height uint32, scale float32) ([]byte, error) {
	if err := ensureCompiled(ctx); err != nil {
		return nil, err
	}

	cfg := wazero.NewModuleConfig().
		WithName("").
		WithStartFunctions("_initialize").
		WithStdout(io.Discard).
		WithStderr(io.Discard)

	mod, err := runtime.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}
	defer mod.Close(ctx)

	alloc := mod.ExportedFunction("alloc")
	dealloc := mod.ExportedFunction("dealloc")
	renderFn := mod.ExportedFunction("render")
	if alloc == nil || dealloc == nil || renderFn == nil {
		return nil, fmt.Errorf("wasm module is missing an expected export")
	}
	mem := mod.Memory()

	put := func(src []byte) (uint32, error) {
		if len(src) == 0 {
			return 0, nil
		}
		res, err := alloc.Call(ctx, uint64(len(src)))
		if err != nil {
			return 0, fmt.Errorf("alloc: %w", err)
		}
		ptr := uint32(res[0])
		if !mem.Write(ptr, src) {
			return 0, fmt.Errorf("write %d bytes out of range", len(src))
		}
		return ptr, nil
	}

	svgPtr, err := put(svg)
	if err != nil {
		return nil, err
	}
	defer dealloc.Call(ctx, uint64(svgPtr), uint64(len(svg)))

	// Fonts framed as repeated [u32 LE len][bytes], matching the Rust shim.
	var fontBlob []byte
	for _, f := range fonts {
		var hdr [4]byte
		binary.LittleEndian.PutUint32(hdr[:], uint32(len(f)))
		fontBlob = append(fontBlob, hdr[:]...)
		fontBlob = append(fontBlob, f...)
	}
	fontsPtr, err := put(fontBlob)
	if err != nil {
		return nil, err
	}
	if len(fontBlob) > 0 {
		defer dealloc.Call(ctx, uint64(fontsPtr), uint64(len(fontBlob)))
	}

	outLenRes, err := alloc.Call(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("alloc out_len: %w", err)
	}
	outLenPtr := uint32(outLenRes[0])
	defer dealloc.Call(ctx, uint64(outLenPtr), 4)

	res, err := renderFn.Call(ctx,
		uint64(svgPtr), uint64(len(svg)),
		uint64(fontsPtr), uint64(len(fontBlob)),
		uint64(width), uint64(height),
		api.EncodeF32(scale),
		uint64(outLenPtr),
	)
	if err != nil {
		return nil, fmt.Errorf("render trapped: %w", err)
	}
	pngPtr := uint32(res[0])

	lenBytes, ok := mem.Read(outLenPtr, 4)
	if !ok {
		return nil, fmt.Errorf("read out_len out of range")
	}
	pngLen := binary.LittleEndian.Uint32(lenBytes)
	if pngPtr == 0 || pngLen == 0 {
		return nil, fmt.Errorf("render failed: invalid SVG, unsupported feature, or bad size")
	}

	view, ok := mem.Read(pngPtr, pngLen)
	if !ok {
		return nil, fmt.Errorf("read PNG out of range")
	}
	out := make([]byte, pngLen)
	copy(out, view)
	dealloc.Call(ctx, uint64(pngPtr), uint64(pngLen))
	return out, nil
}
