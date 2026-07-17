<!-- Edit here: this is the MarkdownDescription source for the burnham javapropertiesencode function. docs/functions/javapropertiesencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a flat string-keyed object as `key=value` lines in alphabetical key order, ready to write to disk. Numeric and boolean values are stringified; nested objects and lists are not allowed.

Keys and values are escaped according to `java.util.Properties` rules:

- `\` is backslash-escaped in both keys and values. `=`, `:`, `#`, and `!` are backslash-escaped in keys only; they are left literal in values. In a key, every space and tab is escaped; in a value, only a leading space is.
- In values, `\n`, `\r`, and `\t` use their short backslash escapes. In keys, `\n` and `\r` do too, but a tab is escaped as a backslash followed by a literal tab (which still round-trips through `java.util.Properties`). Every other control character, and every non-ASCII character, is emitted as a `\uXXXX` escape (a surrogate pair above U+FFFF) for portability across legacy ISO-8859-1 readers.

Output is hand-formatted rather than written via the magiconair/properties library writer, so keys are sorted (the library preserves insertion order) and the output has no leading metadata block.