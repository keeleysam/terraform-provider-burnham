Renders a [Typst](https://typst.app) document to PNG images, one per page, returned as a list of base64-encoded strings (a single-page document is just `result[0]`). Set the resolution with the `ppi` option (pixels per inch; default 144).

Pass structured data through the `inputs` option: it is exposed to the document as `sys.inputs` as native Typst values, so the document reads `sys.inputs.customer.name` directly with no decoding. Use `files` (a map of path to base64-encoded content) for `#import`ed modules and `#image` assets, and `fonts` (a list of base64-encoded fonts) for families beyond the bundled Noto and Liberation sets.

Typst runs as WebAssembly under a pure-Go runtime, so the provider stays CGO-free and the raster output is byte-identical across operating systems and architectures. The one exception is deliberate: a document that calls a non-deterministic Typst builtin such as `datetime.today()` will produce different output on different days.
