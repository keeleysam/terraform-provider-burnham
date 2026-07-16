Parses an [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) string and returns it as the HCL data tree that `oelencode` consumes, so `provider::burnham::oelencode(provider::burnham::oeldecode(expr))` round-trips to the canonical form of `expr`.

This is primarily a tool for testing and for migrating hand-written expressions into the data model.

The decoding mirrors the `oelencode` surface:

- References decode to `{ ident = "..." }`.
- Operators decode to their operator keys: comparisons and `+` use the symbolic token, boolean logic uses `AND`/`OR`, negation uses `!`, and the ternary uses `cond`.
- Calls decode to the `call` forms.
- A dotted path that embeds a group-membership method hop, which has no direct surface form, decodes to a `{ raw = "..." }` escape that `oelencode` re-parses.

The return is a dynamic value, and list literals decode to Terraform tuples (heterogeneous), which `oelencode` accepts on the way back. Covers the full documented grammar. Backed by [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser).