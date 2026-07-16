Returns a Nano ID string derived deterministically from `seed` via HMAC-SHA-256 in counter mode.

The same `seed` always returns the same ID, which makes it a good fit for stable, plan-time identifiers that don't churn on re-apply:

```
nanoid("prod/api-gateway") → "TUgaDb-aFSbMx3UFK6Spd"
```

The default alphabet is the 64-character URL-safe set: underscore, hyphen, then `0-9`, `A-Z`, `a-z` (the literal string `_-0123456789A...Za...z`). This is the same character set the upstream [nanoid](https://github.com/ai/nanoid) reference uses, listed here in sorted order rather than upstream's scrambled order. The default `size` is 21 characters. Both can be overridden via the optional `options` object:

- `alphabet` (string): the alphabet to draw from. Must be non-empty, contain no duplicate runes, and hold at most 256 codepoints. Any Unicode is accepted; bytes are interpreted as a UTF-8 string and you get one alphabet *codepoint* per output position, so a 64-codepoint alphabet still yields a 21-character (= 21-codepoint) ID even if some characters are multi-byte.
- `size` (number): output length in codepoints. Must be in `[1, 1024]`.

~> **Note:** This function is a derivation, not a CSPRNG. Outputs leak nothing about the seed, but two callers seeded with the same secret produce the same ID. Use it for naming, not for credentials.