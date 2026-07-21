package resvg

// The wasm module is built from the Rust shim under rust/ and is gitignored, so
// it must be produced before the package will compile. Run `go generate ./...`
// (requires the Rust toolchain and the wasm32-wasip1 target:
// `rustup target add wasm32-wasip1`).
//
// --locked forces the build to honor the committed Cargo.lock rather than
// re-resolving dependency versions, so the wasm bytes stay reproducible across
// machines and over time, and Dependabot's Cargo.lock bumps are the only thing
// that changes deps.
//
//go:generate sh -c "cd rust && cargo build --release --locked --target wasm32-wasip1 && cp target/wasm32-wasip1/release/svg_render.wasm ../svg_render.wasm"
