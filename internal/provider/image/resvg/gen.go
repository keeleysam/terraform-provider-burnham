package resvg

// The wasm module is built from the Rust shim under rust/ and is gitignored, so
// it must be produced before the package will compile. Run `go generate ./...`
// (requires the Rust toolchain and the wasm32-wasip1 target:
// `rustup target add wasm32-wasip1`).
//
//go:generate sh -c "cd rust && cargo build --release --target wasm32-wasip1 && cp target/wasm32-wasip1/release/svg_render.wasm ../svg_render.wasm"
