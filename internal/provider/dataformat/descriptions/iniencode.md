Encodes a Terraform object as an INI string. The input must be a two-level map: section names at the outer level, key/value pairs at the inner level. Keys under the empty-string key (`""`) are rendered as global keys before any `[section]` header.

All values are converted to strings; sections and keys are written in alphabetical order for deterministic output.

~> **Note:** A value containing a newline fails the plan, since INI has no standard way to represent one.

**Common uses:** generating legacy application config files via `local_file`, or rendering INI snippets to be assembled into a larger config through `templatefile`.