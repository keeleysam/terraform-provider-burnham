Re-encodes `s` under the named [Unicode Normalization Form](https://unicode.org/reports/tr15/) so that strings which look identical also compare equal byte-for-byte. This fixes the classic "looks the same, doesn't compare equal" bug: browsers, macOS, and rich-text editors often hand you NFD-encoded text, while most server-side data is NFC. For most use cases the right call is `unicode_normalize(s, "NFC")`.

The `form` argument selects one of the four canonical forms:

- `"NFC"`: Canonical Composition, the most common server-side form.
- `"NFD"`: Canonical Decomposition.
- `"NFKC"`: Compatibility Composition, collapses ligatures and width variants.
- `"NFKD"`: Compatibility Decomposition.

~> **Note:** The two form families behave differently once the result flows through HCL:

- `"NFC"` and `"NFKC"` (composed) round-trip correctly.
- `"NFD"` and `"NFKD"` (decomposed) do not survive a normal HCL expression: Terraform's value-handling layer (cty) re-normalizes every string to NFC at expression boundaries, so a decomposed result is silently re-composed to NFC the moment it flows into another HCL expression, including `output` blocks.

The decomposed forms are therefore useful only for consumers that ingest the *exact* function return value before Terraform sees it (for example when feeding into another Burnham function within the same expression, or when the byte representation is captured before cty touches it).

Backed by [`golang.org/x/text/unicode/norm`](https://pkg.go.dev/golang.org/x/text/unicode/norm), the canonical Go implementation.