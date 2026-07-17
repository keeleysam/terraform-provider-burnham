<!-- Edit here: this is the MarkdownDescription source for the burnham btoe function. docs/functions/btoe.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a binary key as a sequence of short English words per [RFC 1751](https://www.rfc-editor.org/rfc/rfc1751) ("A Convention for Human-Readable 128-bit Keys"). `btoe` is the RFC's own name for this direction (*bytes-to-english*); `etob` reverses it.

Each 64-bit block of the key becomes six words drawn from a fixed 2048-word dictionary, with two parity bits appended so `etob` can catch a transcription error on the way back. The classic use is reading a key or S/Key one-time password aloud, but it works as a general human-readable encoding for any key material.

The input is a hex string whose decoded length is a **non-zero multiple of 8 bytes** (64-bit blocks); whitespace in the hex is ignored, so `"EB33 F77E E73D 4053"` is accepted. A 128-bit key yields 12 words.

```
btoe("EB33F77EE73D4053")
→ "TIDE ITCH SLOW REIN RULE MOT"
```