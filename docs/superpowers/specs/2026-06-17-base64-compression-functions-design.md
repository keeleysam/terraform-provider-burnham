# Design: `base64zopfli` and `base64brotli` provider functions

Date: 2026-06-17
Status: approved (pending written-spec review)

## Motivation

Modules that bundle many boot scripts into EC2 `user_data` via `base64gzip(jsonencode({...}))` repeatedly hit EC2's 16 KiB raw `user_data` limit. These two provider-defined functions add headroom without otherwise changing the pipeline. They are deliberately complementary, not redundant:

- **`base64zopfli`** — a drop-in replacement for `base64gzip` that uses Zopfli's iterative DEFLATE encoder. Output is a standard gzip member (RFC 1952 / RFC 1951) that any `gunzip`/`zcat`/`compress/gzip` decoder accepts, so consumers change nothing. Win is small (~2%) but free.
- **`base64brotli`** — Brotli (RFC 7932). ~8–10% smaller on text-heavy payloads, but requires a `brotli` decompressor on the consuming side (one-time `apt-get install -y brotli` or AMI inclusion).

`base64gzip` (a Terraform built-in) stays the standards-baseline default; these are opt-in alternatives. Source spec: `/Users/samuel/Desktop/compression-provider-functions-spec.md`.

## Scope

In scope: exactly two functions — `base64zopfli` and `base64brotli`. Out of scope: `base64zstd` / `base64xz` / `base64bzip2` (the spec's design notes evaluate and defer/reject these), custom dictionaries, streaming, and replacing `base64gzip`.

## Hard constraints

- **Pure Go, `CGO_ENABLED=0`.** Burnham releases ~15 cross-compiled, statically-linked, reproducible targets (freebsd/windows/linux/darwin × {amd64,386,arm,arm64}) from a single CI runner via `.goreleaser.yml`. cgo would break single-runner cross-compilation, couple binaries to a platform libc (registry users run these prebuilt binaries on arbitrary machines), and complicate reproducible builds. This was reviewed explicitly and reaffirmed: the cgo-only upside here (Google's upstream zopfli; a brotli encoder that honors the `mode` hint) is marginal and not worth the distribution cost. Both functions use pure-Go libraries.
- **Determinism.** Identical input + identical options must produce byte-identical output, every plan, across processes. Terraform plan stability depends on it. Pinned by tests.

## Libraries

| Function | Library | Why |
|---|---|---|
| `base64zopfli` | `github.com/foobaz/go-zopfli/zopfli` | Pure-Go Zopfli port. Supports block-splitting (where Zopfli's win over gzip comes from). `DefaultOptions()` already matches the spec: `NumIterations=15, BlockSplitting=true, BlockSplittingLast=false, BlockType=DYNAMIC`. Exposes `DeflateCompress` (raw DEFLATE) — needed to assemble a spec-exact gzip container (see below). |
| `base64brotli` | `github.com/andybalholm/brotli` | Pure-Go Brotli encoder used by Caddy and others; competitive with the C encoder at quality 11. `WriterOptions{Quality, LGWin}`. |

## `base64zopfli(input, ...options) → string`

### Signature & options

- `input` (string, required) — arbitrary byte string. Matches `base64gzip`'s input type.
- `options` (optional object, at most one): `{ iterations = number }`.
  - `iterations`: default **15**, validated to **[1, 100000]**. Pure CPU/size tradeoff; output is always valid DEFLATE.

### Encoding

The library's own `GzipCompress` writes `OS=3` (Unix), but the spec wants `OS=255` (unknown — most-portable, avoids leaking an implementation detail). So we compress to **raw DEFLATE** with `zopfli.DeflateCompress` and assemble the gzip container by hand for full byte-level control:

```
header  = 1f 8b 08 00  00 00 00 00  02 ff
          │  │  │  │    └ MTIME = 0 (4B) ┘  │  └ OS  = 255 (unknown)
          │  │  │  └ FLG = 0 (no FNAME/FEXTRA/FHCRC/FTEXT)
          │  │  └ CM  = 8 (DEFLATE)
          └──┴ ID1 ID2 = 0x1f 0x8b
                                          └ XFL = 2 (max compression)
body    = <zopfli raw DEFLATE bytes>
trailer = CRC32(input) little-endian (4B) ++ uint32(len(input) mod 2^32) little-endian (4B)
```

`hash/crc32` (IEEE) for CRC; `encoding/binary` (LittleEndian) for both trailer fields. The whole gzip member is then `base64.StdEncoding`-encoded.

Internal Zopfli options: `DefaultOptions()` with `NumIterations` overridden by `iterations`. `BlockSplitting` stays true; do not disable.

### Determinism

`MTIME=0` (hardcoded), all other header fields constant, Zopfli is deterministic for a given input + iterations. Pinned by a compress-twice byte-equality test and a header-bytes assertion.

## `base64brotli(input, ...options) → string`

### Signature & options

- `input` (string, required) — arbitrary byte string.
- `options` (optional object, at most one): `{ quality = number, lgwin = number }`.
  - `quality`: default **11**, range **[0, 11]**. Default 11 because `user_data` is compressed once at plan time and decompressed many times.
  - `lgwin`: default **22** (4 MiB window), range **[10, 24]** per RFC 7932 §9.1.

### `mode` is intentionally omitted

The spec lists a `mode` option (`text`/`generic`/`font`, default `text`). It is **not** exposed, because the pure-Go encoder cannot honor it meaningfully:

- Grepping every read of the mode field in `andybalholm/brotli` shows exactly one use — `encode.go:511-513`, `if params.mode == modeFont { ... }`. `modeText` is defined as a constant but **never read anywhere**. The C reference encoder uses mode to switch literal-context modeling (UTF-8 vs signed); this port makes that decision independently of the hint.
- Therefore `text` and `generic` produce **byte-identical** output here, and the only mode that changes anything (`font`) is unreachable through the public `WriterOptions` API.
- Exposing `mode` would mean shipping a knob that is a no-op for `text`/`generic` and a lie for `font`. Omitting it leaves zero compression on the table (`text == generic` in this encoder). The function's `MarkdownDescription` documents this decision so the "why" is on record. True mode support is a future `base64brotli` revision if a pure-Go encoder ever exposes it.

### Encoding & determinism

Map directly to `brotli.WriterOptions{Quality: quality, LGWin: lgwin}`, write `input`, base64-encode. The encoder is inherently deterministic; pinned by a compress-twice byte-equality test.

## Errors (both functions)

Plan-time `function.NewArgumentFuncError(1, …)`, reusing `optionsutil.SingleOptionsObject` (rejects non-object / >1 options) and `optionsutil.NumberAttrToInt`:

- options argument that isn't an object literal.
- more than one options object.
- unknown option key — message lists the supported keys (matching the `nanoid` pattern).
- out-of-range numeric (`iterations`, `quality`, `lgwin`) — message names the field and the valid range.

Compression failure (extremely unlikely for valid string input) surfaces the underlying library error.

## Placement (new "Compression" family — 9th family)

```
internal/provider/compression/
  functions.go      // package doc + Functions() → NewBase64ZopfliFunction, NewBase64BrotliFunction
  base64zopfli.go
  base64brotli.go
  base64zopfli_test.go   // package-level unit tests
  base64brotli_test.go
```

Wiring:
- `internal/provider/provider.go` — add `compression.Functions()` to `Functions()`; update the `Schema` description (it enumerates the families).
- `cmd/gendoctemplates/main.go` — add `{"Compression", compression.Functions()}` to the `families` slice and import the package. This is the single source of truth for the docs subcategory; `go generate ./...` then emits `templates/functions/base64{zopfli,brotli}.md.tmpl` (gitignored) and `docs/functions/base64{zopfli,brotli}.md` (committed).
- `templates/index.md.tmpl` and `README.md` — bump "eight families" → "nine", add a Compression section/table.

## Testing — two layers (in-process only)

No shelling out to system binaries — the rest of the repo verifies entirely in-process, and we match that. Decompression in tests happens in Go: `base64zopfli` output through the standard library's `compress/gzip` reader (an RFC 1952 decoder entirely independent of the Zopfli encoder, so it genuinely cross-checks header conformance), and `base64brotli` output through `brotli.NewReader`.

1. **Package unit tests** (`internal/provider/compression/*_test.go`):
   - Round-trip: `base64zopfli` output → `compress/gzip` reader → original; `base64brotli` output → `brotli.NewReader` → original.
   - Determinism: compress the same `(input, options)` twice, assert `bytes.Equal`.
   - Empty input: both must succeed and decompress to `""`.
   - Large input: a ~3 MiB payload round-tripped in-process (window-size / buffer sanity).
   - Conformance: zopfli's full 10-byte header pins `1f 8b 08`, `MTIME=0`, `XFL=2`, `OS=255`; trailer CRC32 + ISIZE checked against `hash/crc32` and `len(input)`.
   - Ratio: both beat (or, for zopfli, match-or-beat) `gzip -9` on representative text.
   - Option matrix: each tested `iterations` / `quality` / `lgwin` round-trips.
2. **Acceptance tests** (`internal/provider/acceptance_compression_test.go`, via `runOutputTest` / `runErrorTest`):
   - Default-equals-explicit equality (proves defaults are 15 / 11+22 and pins determinism through the full `Run` path).
   - Empty input produces a valid base64 string.
   - Every documented error path (bad `iterations`, bad `quality`, bad `lgwin`, unknown option key including `mode`, non-object options).

## Docs, examples, changelog

- `examples/functions/base64zopfli/function.tf`, `examples/functions/base64brotli/function.tf` — runnable snippets (drop-in `base64gzip` swap; explicit-options variant).
- `go generate ./...` to regenerate `docs/functions/*.md`.
- README Compression family section + the eight-families → nine-families intro edits.
- `CHANGELOG.md` entry.

## Deviations from the source spec

1. **cgo libraries not used** — pure-Go `foobaz/go-zopfli` and `andybalholm/brotli` instead, per the `CGO_ENABLED=0` constraint. No effect on output format, RFC compliance, or determinism.
2. **brotli `mode` omitted** — the pure-Go encoder can't honor it (see above). The only documented-but-unbuildable part of the spec.

Everything else follows the spec exactly, including `base64zopfli`'s `OS=255`/`MTIME=0`/`XFL=2` gzip header, the option ranges and defaults, and the determinism requirement.
