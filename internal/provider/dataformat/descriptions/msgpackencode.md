<!-- Edit here: this is the MarkdownDescription source for the burnham msgpackencode function. docs/functions/msgpackencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a Terraform value as [MessagePack](https://msgpack.org/) and returns the result as a standard base64 string. Object keys are written in sorted order for stable output. Whole-number floats are emitted as integers (matching the conventions of `jsonencode` here).

**Common uses:** generating msgpack payloads to seed Redis fixtures, write to disk via `local_file`, or feed external tooling.