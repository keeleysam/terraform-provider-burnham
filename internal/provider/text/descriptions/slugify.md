Returns a URL-safe slug derived from `s`. Lowercases the result, transliterates non-ASCII characters into their nearest ASCII equivalent (`café` → `cafe`, `Москва` → `moskva`, `北京` → `bei-jing`), strips remaining punctuation, and joins runs of word characters with hyphens.

```
slugify("Café au Lait №3")  → "cafe-au-lait-3"
slugify("Hello, World!")     → "hello-world"
```

Options object:

- `language` (string): ISO 639-1 hint for transliteration (e.g. `"en"`, `"de"`, `"ja"`). The default heuristic produces good output for Latin-script input; pick a language to handle non-Latin input correctly. Library list of supported codes: see [gosimple/slug](https://github.com/gosimple/slug).
- `separator` (string): the joiner between words. Default `"-"`.
- `lowercase` (bool): lowercase the result. Default `true`.

Different from Terraform's `replace()` + `lower()` and from corefunc's case-conversion functions: `slugify` does **transliteration**, not just case folding.