Evaluates an [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) expression against a supplied context and returns the result, for previewing or testing a group rule or profile mapping at plan time.

~> **Note:** This is a local approximation, not Okta's engine. Real evaluation happens server-side against live data.

The optional second argument is a context object with these keys:

- `user`: an object resolved by `user.<attr>` paths.
- `group_ids`: a list of group IDs the user is a member of, for `isMemberOfGroup` and `isMemberOfAnyGroup`.
- `groups`: group metadata keyed by ID, each with a nested `profile.name`, for the `isMemberOfGroupName` family.
- `strict`: a bool. When true, a path access to an attribute absent from `user` errors instead of resolving to null.

For example, `provider::burnham::oelevaluate("user.department == \"Sales\"", { user = { department = "Sales" } })` returns `true`.

Evaluation covers the group-rule subset:

- Literals, comparisons, and boolean logic.
- The ternary and `+` operators.
- The `String`, `Arrays`, `Convert`, `Iso3166Convert`, and `Groups` class functions.
- The bare `isMemberOf*` group builtins.
- `user.<attr>` paths.

Expressions using receiver method calls (`user.getInternalProperty(...)`, the Identity Engine method dialect, `user.isMemberOf({...})`, `getGroups`), projection, indexing, Elvis, or `matches` parse but are not evaluated and return an error. Use `oelencode`, `oelvalidate`, or `oelformat` for those.

Backed by [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser).