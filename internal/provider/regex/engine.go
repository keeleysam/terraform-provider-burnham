// Package regex provides PCRE-flavored regular-expression functions (pcre_*)
// backed by the Rust fancy-regex engine compiled to wasm32-wasip1 and run under
// the pure-Go wazero runtime. Unlike Terraform core's RE2-based regex functions,
// these support backreferences and lookaround. Pure, CGO-free, and deterministic.
package regex

import (
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed pcre.wasm
var wasmBytes []byte

// Operation selectors, matching the Rust shim's `run` op argument.
const (
	opMatch    = 0
	opCaptures = 1
	opFindAll  = 2
	opReplace  = 3
	opSplit    = 4
)

/*
EngineError is a user-caused error reported by the regex engine itself: an invalid pattern or a
runtime failure such as the backtrack limit tripping on a catastrophic pattern. It carries the
engine's own message. The function layer maps it to an argument diagnostic (the pattern argument),
whereas every other error runOp can return is an internal fault (a wasm trap, a missing result, a
decode failure) that should surface as a general function error rather than blaming the caller's
input.
*/
type EngineError struct{ Msg string }

func (e *EngineError) Error() string { return e.Msg }

var (
	initMu   sync.Mutex
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
)

// ensureCompiled compiles the wasm module once per process; each call
// instantiates its own module (its own memory), so calls don't share state.
//
// A failed setup is deliberately not cached. Terraform calls provider functions
// concurrently and may hand the very first call an already-cancelled context, so
// latching that first error (as a sync.Once would) could disable the whole regex
// family for the life of the process even though a later call would succeed. The
// mutex-guarded check is uncontended once compiled is set, and cheap next to the
// per-call module instantiation, so a later call simply retries.
func ensureCompiled(ctx context.Context) error {
	initMu.Lock()
	defer initMu.Unlock()
	if compiled != nil {
		return nil
	}
	// WithCloseOnContextDone makes wazero honor context cancellation during wasm
	// execution: if Terraform cancels the plan, an in-flight call is interrupted
	// rather than running to completion on a pinned core.
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

// runOp runs one regex operation and returns the decoded result value (the "v"
// field of the JSON envelope), or an error if the pattern or operation failed.
func runOp(ctx context.Context, op uint32, pattern, input, replacement string) (json.RawMessage, error) {
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
	runFn := mod.ExportedFunction("run")
	if alloc == nil || dealloc == nil || runFn == nil {
		return nil, errors.New("wasm module is missing an expected export")
	}
	mem := mod.Memory()

	// put copies s into wasm memory, returning (ptr, len). The caller deallocs.
	put := func(s string) (uint32, uint32, error) {
		if len(s) == 0 {
			return 0, 0, nil
		}
		res, err := alloc.Call(ctx, uint64(len(s)))
		if err != nil {
			return 0, 0, fmt.Errorf("alloc: %w", err)
		}
		ptr := uint32(res[0])
		if !mem.Write(ptr, []byte(s)) {
			return 0, 0, errors.New("write out of range")
		}
		return ptr, uint32(len(s)), nil
	}

	patPtr, patLen, err := put(pattern)
	if err != nil {
		return nil, err
	}
	defer dealloc.Call(ctx, uint64(patPtr), uint64(patLen))
	inpPtr, inpLen, err := put(input)
	if err != nil {
		return nil, err
	}
	defer dealloc.Call(ctx, uint64(inpPtr), uint64(inpLen))
	repPtr, repLen, err := put(replacement)
	if err != nil {
		return nil, err
	}
	defer dealloc.Call(ctx, uint64(repPtr), uint64(repLen))

	outLenRes, err := alloc.Call(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("alloc out_len: %w", err)
	}
	outLenPtr := uint32(outLenRes[0])
	defer dealloc.Call(ctx, uint64(outLenPtr), 4)

	res, err := runFn.Call(ctx,
		uint64(op),
		uint64(patPtr), uint64(patLen),
		uint64(inpPtr), uint64(inpLen),
		uint64(repPtr), uint64(repLen),
		uint64(outLenPtr),
	)
	if err != nil {
		return nil, fmt.Errorf("regex engine trapped: %w", err)
	}
	outPtr := uint32(res[0])

	lenBytes, ok := mem.Read(outLenPtr, 4)
	if !ok {
		return nil, errors.New("read out_len out of range")
	}
	outLen := binary.LittleEndian.Uint32(lenBytes)
	if outPtr == 0 || outLen == 0 {
		return nil, errors.New("regex engine returned no result")
	}
	view, ok := mem.Read(outPtr, outLen)
	if !ok {
		return nil, errors.New("read result out of range")
	}
	envelope := make([]byte, outLen)
	copy(envelope, view)
	dealloc.Call(ctx, uint64(outPtr), uint64(outLen))

	var env struct {
		V json.RawMessage `json:"v"`
		E string          `json:"e"`
	}
	if err := json.Unmarshal(envelope, &env); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	if env.E != "" {
		return nil, &EngineError{Msg: env.E}
	}
	// A well-formed envelope always carries exactly one of "v" or "e". The no-match cases encode an
	// explicit JSON null ({"v":null}), which unmarshals to a non-nil RawMessage of "null", so a nil V
	// here means the shim emitted neither field: treat that as an internal fault rather than silently
	// returning (nil, nil), which would surface far downstream as a confusing decode error.
	if env.V == nil {
		return nil, errors.New("regex engine returned an empty envelope")
	}
	return env.V, nil
}
