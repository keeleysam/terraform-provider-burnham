Authorizes a request against a [Cedar](https://www.cedarpolicy.com) policy document and returns the decision, for previewing or unit-testing authorization policies at plan time.

Because it uses [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar, the decision comes from Cedar's own evaluation engine rather than an approximation (Amazon Verified Permissions is built on the same engine).

The second argument is the request object:

- `principal`, `action`, `resource`: each an object with `type` and `id`, for example `{ type = "User", id = "alice" }`.
- `context`: an optional plain attribute record such as `{ mfa = true }`, referenced in policies as `context.<key>`.
- `entities`: an optional list supplying the attributes and hierarchy the decision resolves against.

Each `entities` element uses the Cedar entities shape:

```hcl
{
  uid     = { type = ..., id = ... }
  attrs   = { ... }
  parents = [{ type = ..., id = ... }, ...]
}
```

Returns an object with:

- `decision`: `"allow"` or `"deny"`.
- `reasons`: the ids of the policies that determined the decision.
- `errors`: any evaluation errors.

~> **Note:** A policy with no `@id` annotation is numbered `policy0`, `policy1`, and so on in document order. Add `@id("...")` to get stable names in `reasons`.