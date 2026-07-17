<!-- Edit here: this is the MarkdownDescription source for the burnham hujsondecode function. docs/functions/hujsondecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses a HuJSON ([JSON With Commas and Comments / JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html)) string into a Terraform value. Standard JSON is also accepted, since HuJSON is a strict superset.

Comments (`//` line and `/* */` block) are stripped during parsing; trailing commas are tolerated. Object keys become object members, arrays become tuples, and numbers preserve precision via `json.Number`.

**Common uses:** parsing [Tailscale ACL policies](https://tailscale.com/kb/1018/acls), VS Code-style configuration files (`tsconfig.json`, `.vscode/settings.json`), or any human-edited JSON variant that allows comments.