<!-- Edit here: this is the MarkdownDescription source for the burnham cbordecode function. docs/functions/cbordecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes [CBOR](https://www.rfc-editor.org/rfc/rfc8949) ([RFC 8949](https://www.rfc-editor.org/rfc/rfc8949)) bytes into a Terraform value. Provide the bytes as a standard base64 string, since HCL strings are UTF-8 only.

Type mapping:

- CBOR maps with string keys become objects (maps with non-string keys are an error)
- arrays become tuples
- integers and floats become numbers
- byte strings become standard base64 strings
- tag-0 and tag-1 datetimes become [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339) strings
- bignum tags (2 and 3) become full-precision numbers (Terraform's number type uses arbitrary-precision big floats)

Backed by [fxamacker/cbor](https://github.com/fxamacker/cbor), an RFC 8949 conforming implementation.

**Common uses:** consuming CBOR-encoded payloads from IoT/CoAP gateways, COSE signed objects, or any binary structured-data feed where compactness matters more than human readability.