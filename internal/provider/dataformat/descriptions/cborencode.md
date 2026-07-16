Encodes a Terraform value as [CBOR](https://www.rfc-editor.org/rfc/rfc8949) ([RFC 8949](https://www.rfc-editor.org/rfc/rfc8949)) and returns the result as a standard base64 string. Output uses CBOR's [Core Deterministic Encoding](https://www.rfc-editor.org/rfc/rfc8949#section-4.2.1): definite-length items, sorted map keys, and shortest-form integers, so the same input produces byte-identical output.

Whole-number floats are emitted as integers (matching the conventions of `jsonencode` here). Strings are encoded as CBOR text strings; the function does not synthesize byte strings or tagged values from HCL inputs.

Backed by [fxamacker/cbor](https://github.com/fxamacker/cbor).

**Common uses:** generating CBOR fixtures for IoT services, COSE-style payloads, or any binary feed that benefits from a deterministic encoding.