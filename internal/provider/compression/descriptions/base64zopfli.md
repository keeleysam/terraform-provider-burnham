Compresses `input` with [Zopfli](https://github.com/google/zopfli)'s iterative DEFLATE encoder and returns the result as a base64-encoded gzip member. A drop-in replacement for Terraform's built-in `base64gzip`: the output is an ordinary [RFC 1952](https://www.rfc-editor.org/rfc/rfc1952) gzip stream that decompresses with any `gunzip` / `zcat` / `compress/gzip` decoder; consumers cannot tell it came from Zopfli rather than `gzip -9`, and nothing on the decompression side has to change.

Zopfli spends much more CPU than zlib searching for a smaller encoding of the same data (typically ~2–5% smaller than `gzip -9` on text). The win is free at the wire: it just makes the plan-time compression slower.

The optional `options` object accepts a single key:

- `iterations` (number): Zopfli optimization passes. Default `15`, range `[1, 100000]`. Higher is smaller, with diminishing returns past ~100. Any value produces valid DEFLATE.

-> **Note:** The gzip header is fixed for deterministic, portable output: `MTIME=0` (never "current time", which would churn every plan), `XFL=2`, `OS=255` (unknown), and no optional flags. The same `input` and options always produce byte-identical output, keeping plans stable.