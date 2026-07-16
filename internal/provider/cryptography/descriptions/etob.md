Decodes a sequence of [RFC 1751](https://www.rfc-editor.org/rfc/rfc1751) English words back into the original key, returning lowercase hex. `etob` is the RFC's own name for this direction (*english-to-bytes*); `btoe` produces the words.

The input is a phrase whose word count is a **non-zero multiple of six** (each six words decode to one 64-bit block). Words are matched case-insensitively and the RFC's `standard()` normalization is applied (`1`→`L`, `0`→`O`, `5`→`S`), so dictation/OCR slips are tolerated. The two parity bits embedded by `btoe` are verified; a phrase that fails the parity check (a likely transcription error) is rejected, as is any word not in the dictionary.

```
etob("TIDE ITCH SLOW REIN RULE MOT")
→ "eb33f77ee73d4053"
```