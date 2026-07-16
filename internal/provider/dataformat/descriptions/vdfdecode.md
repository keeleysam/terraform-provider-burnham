Parses a [Valve Data Format (VDF)](https://developer.valvesoftware.com/wiki/KeyValues) string into a Terraform object. VDF is a nested key-value format used by Steam and the Source engine; the only types are strings and nested objects, so all leaf values come back as strings.

`//` comments are stripped during parsing.

**Common uses:** reading Steam app manifests (`appmanifest_*.acf`), Source engine config files, or any Valve-tooling artifact where the on-disk format is VDF rather than INI/JSON.