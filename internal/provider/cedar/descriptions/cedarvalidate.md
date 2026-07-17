<!-- Edit here: this is the MarkdownDescription source for the burnham cedarvalidate function. docs/functions/cedarvalidate.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `true` if `policies` is a syntactically valid [Cedar](https://www.cedarpolicy.com) policy document, `false` otherwise (an empty document is valid). Unlike `cedarformat`, it does not fail the plan on invalid input, so it suits a boolean check in a `precondition` guarding a hand-written `aws_verifiedpermissions_policy` statement.

Validation is syntactic (parsing); it does not check policies against a schema. Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.