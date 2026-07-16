Evaluates a [jq](https://jqlang.github.io/jq/) program against a Terraform value and returns the program's output stream as a list. jq is the most widely-used JSON query language, and this is the expressive sibling of `jmespath_query` and `jsonpath_query`: it supports pipes, `reduce`, `map`/`select`, string interpolation, object construction, and the rest of the jq language.

Because a jq program is a *stream* (`.[]` emits one result per element, `.a, .b` emits two), `jq` always returns a **list** with one element per value the program produced:

- A program that yields a single value returns a one-element list. Collapse it with `one(provider::burnham::jq(...))` or index the first element.
- A program that yields nothing returns an empty list.

Named bindings are passed through the optional `vars` object and referenced as jq variables. For example, `{ vars = { tier = "prod" } }` binds `$tier`, so the program can `select(.tier == $tier)`.

### Number handling

Non-integers use IEEE-754 `float64` precision (the same as `jmespath_query`). Unlike `jmespath_query`, which floats every number, `jq` preserves integers beyond 2^53 exactly. (`jsonpath_query`, by contrast, passes values through untouched and preserves every number at full precision.)

### Unsupported builtins

- `env` and `$ENV` return an empty object; this function does not expose the host process environment.
- `input` and `inputs` error, because there is no secondary input stream.

### Execution limits

Execution is bounded so a runaway program fails instead of hanging or exhausting memory:

- A program that runs longer than 30 seconds is cancelled and returns an error, guarding against non-terminating programs (jq is Turing-complete).
- A program that emits more than 1,000,000 values fails rather than accumulating an unbounded result.
- A result nested deeper than 1024 levels returns `result exceeds maximum supported nesting depth of 1024`.

~> **Note:** Most jq programs are pure functions of their input, but `now` and `localtime` read the wall clock (and host timezone). Any program deriving from them may produce different output on each plan or apply and will churn the plan, so use them only when you intend that.

Backed by [itchyny/gojq](https://github.com/itchyny/gojq), a pure-Go jq implementation.