package typst

// The wasm engine is built from the Rust shim under rust/ and is gitignored, so
// it must be produced before this package compiles. Run `go generate ./...`
// (requires the Rust toolchain and the wasm32-wasip1 target:
// `rustup target add wasm32-wasip1`).
//
// --locked forces the build to honor the committed Cargo.lock rather than
// re-resolving dependency versions, so the wasm bytes stay reproducible across
// machines and over time, and Dependabot's Cargo.lock bumps are the only thing
// that changes deps.
//
//go:generate sh -c "cd rust && cargo build --release --locked --target wasm32-wasip1 && cp target/wasm32-wasip1/release/typst_engine.wasm ../typst.wasm"
