<!-- Edit here: this is the MarkdownDescription source for the burnham json_merge_patch function. docs/functions/json_merge_patch.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Applies an [RFC 7396](https://www.rfc-editor.org/rfc/rfc7396) JSON Merge Patch to a Terraform value and returns the merged result. Unlike RFC 6902 (JSON Patch), a merge patch *is* a partial document with the same shape as the target, which makes it the right tool for environment overlays and Kubernetes-style strategic-merge-adjacent layering where most of your patch is just "set these fields, remove that one."

The patch is applied by these rules:

- Keys present in the patch override the matching keys in the target.
- A `null` value in the patch deletes the corresponding key from the target.
- Arrays are replaced wholesale; they are not merged element-wise.

-> **Note:** For element-level array edits or `test`-gated operations, use `json_patch` (RFC 6902) instead.

Backed by [evanphx/json-patch](https://github.com/evanphx/json-patch).