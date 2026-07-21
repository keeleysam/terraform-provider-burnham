Exports a [Typst](https://typst.app) document to a single self-contained HTML string: CSS is inlined, vector graphics are embedded as inline SVG, and raster images are embedded as base64 `data:` URLs, so there are no external files. The result is returned as text, not base64.

**Experimental.** HTML is Typst's experimental export target: it produces semantic HTML (headings, paragraphs, lists, tables, links) rather than a pixel-perfect rendering, its feature coverage is still evolving upstream, and layout-precise constructs may not translate. For pixel-faithful output use `typst_pdf`, `typst_png`, or `typst_svg`.

Pass structured data through the `inputs` option: it is exposed to the document as `sys.inputs` as native Typst values, so the document reads `sys.inputs.customer.name` directly with no decoding. Use `files` (a map of path to base64-encoded content) for `#import`ed modules and `#image` assets, and `fonts` (a list of base64-encoded fonts) for extra families.

Typst runs as WebAssembly under a pure-Go runtime, so the provider stays CGO-free and the output is deterministic across operating systems and architectures. The one exception is deliberate: a document that calls a non-deterministic Typst builtin such as `datetime.today()` will produce different output on different days.
