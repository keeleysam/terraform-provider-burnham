Evaluates a [JMESPath](https://jmespath.org/) expression against a Terraform value and returns the matching result. Use it to extract fields from large nested structures (decoded API responses, manifests, configuration trees) without long chains of `try(local.x.foo[0].bar, null)`.

The expression follows the JMESPath specification, including:

- Projections: `[*]`
- Filters: `[?key == 'value']`
- Pipes: `|`
- Functions: `length`, `sort_by`, `to_string`, and the rest
- Multi-select hashes: `{a: foo, b: bar}`

A field or index that does not exist evaluates to `null`; a filter or projection that matches nothing evaluates to an empty list `[]`.

~> **Note:** Numbers are evaluated as IEEE 754 double-precision floats, the only numeric type the JMESPath engine supports, so an integer whose magnitude exceeds 2^53 can come back rounded. If you need to carry such a value through unchanged, select it with a different function (for example `jq`) rather than JMESPath.

Backed by [jmespath-community/go-jmespath](https://github.com/jmespath-community/go-jmespath), the actively-maintained fork of the canonical Go implementation.