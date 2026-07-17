<!-- Edit here: this is the MarkdownDescription source for the burnham jsonpath_query function. docs/functions/jsonpath_query.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Evaluates an [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535.html) JSONPath expression against a Terraform value and returns the list of matching nodes. Use it to extract subsets of nested structures with the standardized IETF JSONPath grammar.

The expression must conform to RFC 9535. Common selectors include:

- Root identifier: `$`
- Name selectors: `$.store.book`
- Wildcard: `$.store.*`
- Descendant segments: `$..price`
- Descendant wildcard (all descendant nodes): `$..*`
- Array slices: `$[0:5]`
- Filters: `$[?@.price < 10]`

The result is always a list of matching values. An expression that matches nothing returns an empty list. To collapse a single-match query to a scalar, use `one(provider::burnham::jsonpath_query(...))` or index the first element.

Backed by [theory/jsonpath](https://github.com/theory/jsonpath), an RFC 9535 conforming implementation.