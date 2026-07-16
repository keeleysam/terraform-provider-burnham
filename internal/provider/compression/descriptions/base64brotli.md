Compresses `input` with [Brotli](https://www.rfc-editor.org/rfc/rfc7932) and returns the result as a base64-encoded brotli stream. Use it when you want the smallest possible blob and can afford a brotli decompressor on the consuming side.

On text-heavy payloads this is ~8–10% smaller than `base64gzip` (and a few percent smaller than `base64zopfli`), at the cost of requiring a brotli decompressor to read it. Decompress with `base64 -d | brotli -d`, or any RFC 7932 decoder (browsers' `Content-Encoding: br`, Python `brotli`, and so on). `brotli -d` ships with every current Linux distro.

The optional `options` object accepts:

- `quality` (number): compression effort. Default `11` (maximum ratio), range `[0, 11]`. Lower is faster with a worse ratio. The default is `11` because inputs are typically compressed once at plan time and decompressed many times, so maximum ratio is usually the right trade-off.
- `lgwin` (number): log₂ of the sliding-window size in bytes (RFC 7932 §9.1). Default `22` (a 4 MiB window), range `[10, 24]`. Increase only for genuinely huge inputs with long-range repetition; decrease only if compress-time memory is constrained.

The brotli encoder `mode` hint (text/generic/font) is intentionally **not** exposed: the pure-Go encoder this provider uses has no mode field to set, so `quality` and `lgwin` are the only knobs that change the output. A `mode` option would be a no-op rather than an honest knob.

-> **Note:** The encoder is deterministic for a given input and options (there is no MTIME-equivalent in the brotli format), so the same `input` and options always produce byte-identical output, keeping plans stable.