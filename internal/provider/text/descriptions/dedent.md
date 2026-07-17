<!-- Edit here: this is the MarkdownDescription source for the burnham dedent function. docs/functions/dedent.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Removes the longest run of leading whitespace common to all non-blank lines, leaving relative indentation intact, the `textwrap.dedent` operation. Terraform ships `indent` but no inverse, so an indented heredoc (cloud-init, a shell script, an embedded YAML/JSON block) comes out indented; `dedent` left-aligns it.

Whitespace-only lines don't count toward the common margin and are normalized to empty. Tabs and spaces must match exactly to be considered common, so don't mix them in the indentation you intend to strip.

```
dedent("    if x:\n        y")
→ "if x:\n    y"
```