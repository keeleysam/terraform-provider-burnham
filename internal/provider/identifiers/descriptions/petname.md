Returns a short, human-friendly identifier composed of dictionary words, the same form `dustinkirkland/golang-petname` and Heroku app names take.

Deterministic: the same `seed` always returns the same petname. The word indices are derived from `HMAC-SHA-256(seed, "burnham/petname")`, which keeps the result stable across plans without leaking anything about the seed.

Word-count patterns match upstream petname:

- 1: `<noun>`, e.g. `"fox"`
- 2: `<adjective>-<noun>`, e.g. `"swift-fox"` *(default)*
- 3: `<adverb>-<adjective>-<noun>`, e.g. `"gently-swift-fox"`
- 4+: extra adverbs stack at the front, e.g. `"calmly-gently-swift-fox"`

Options object:

- `words` (number): word count in `[1, 16]`. Default 2.
- `separator` (string): joiner between words. Default `"-"`.

~> **Note:** Wordlists are short (64 entries each), so a 2-word petname has 64 × 64 = 4096 possible outputs and 3-word has 262144 (64 × 64 × 64). For a high-uniqueness deterministic identifier, prefer `nanoid` or `uuid_v5`; petname is for *readable* identifiers, not collision-resistant ones.