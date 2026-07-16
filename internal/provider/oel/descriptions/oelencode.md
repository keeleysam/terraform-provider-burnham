Builds an [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) (OEL) string from a structured HCL value, so you assemble expressions from Terraform data (variables, `for` expressions, `merge`, `concat`) without string templating or manual quote escaping.

The result is a canonical string suitable for `okta_group_rule.expression_value`, `okta_profile_mapping` mapping expressions, `okta_app_signon_policy_rule.custom_expression`, and Okta Identity Governance policy conditions.

### Leaves

- Bare strings, numbers, booleans, and `null` are literals.
- A bare list is an OEL array literal (`{1, 2, 3}`).
- A reference (an attribute path such as `user.department` or `appuser.email`) is the only marked leaf: `{ ident = "user.department" }`.

### Operators

Operators are OEL tokens or friendly aliases, applied to an operand list:

- Comparison: symbolic tokens `==`, `!=`, `<`, `>`, `<=`, `>=`, or the aliases `eq`, `ne`, `lt`, `gt`, `le`, `ge`; each takes a two-element operand list (`{ "==" = [a, b] }` or `{ eq = [a, b] }`).
- Boolean: `and` and `or` are n-ary (the uppercase `AND`/`OR` that `oeldecode` emits, and the symbolic `&&`/`||`, are also accepted, all normalized to `AND`/`OR`), and `not` takes a single operand (`{ not = a }`).
- Concatenation: `+` is n-ary.
- Ternary: `{ cond = [test, ifTrue, ifFalse] }` (alias `{ "?:" = [...] }`).
- Elvis: `{ elvis = [value, default] }`.
- Regex: `{ matches = [subject, pattern] }`.

~> **Note:** `matches` is deprecated by Okta. It is kept for round-tripping existing expressions.

-> **Note:** `!` and `not` are interchangeable on input, and logical operators are always emitted in the `AND`/`OR` keyword form.

### Calls

A `call` takes one of three forms:

- A namespaced class method: `{ call = { class = "String", method = "startsWith", args = [ { ident = "user.firstName" }, "prod-" ] } }`.
- A bare function: `{ call = { function = "isMemberOfAnyGroup", args = ["00g..."] } }`. This covers the `isMemberOf*` group builtins and any other bare function such as `substringBefore` or `getManagerUser`.
- A receiver method call: `{ call = { target = { ident = "user" }, method = "getInternalProperty", args = ["status"] } }`.

The receiver form also expresses the Identity Engine method dialect (`{ call = { target = { ident = "user.profile.firstName" }, method = "toUpperCase" } }`) and object-argument membership (`user.isMemberOf({...})`, with the object built via `map`).

### Access and structure

- `{ select = { operand = <expr>, field = "firstName" } }` is a property access on a non-identifier receiver (a plain path should use `ident`).
- `{ index = { base = <expr>, index = <expr> } }` is `base[index]`.
- `{ project = { base = <expr>, expr = <expr> } }` is the collection projection `base.![expr]`.
- `{ map = [ { key = "group.profile.name", value = "X" }, { key = "operator", value = "EXACT" } ] }` is an ordered object literal (`{"group.profile.name": "X", "operator": "EXACT"}`).

### Escape hatch

`{ raw = "<okta el>" }` embeds a hand-written OEL fragment. It is parsed, and so validated, like any other node.

Backed by [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser), which handles operator precedence, parenthesization, and quoting.

-> **Note:** The output is always parsed back before it is returned, so `oelencode` can never produce a syntactically invalid expression. The result is canonical, byte-identical to what `oelformat` produces from the same expression.