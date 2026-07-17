<!-- Edit here: this is the MarkdownDescription source for the burnham applestringsdecode function. docs/functions/applestringsdecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses an Apple [`.strings`](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPInternational/MaintaingYourOwnStringsFiles/MaintaingYourOwnStringsFiles.html) localization file body into a flat string-to-string object. Both UTF-8 and UTF-16 (with BOM) inputs are auto-detected. `//` and `/* */` comments are tolerated and skipped.

Each entry follows `"key" = "value";` with C-style escapes inside the quoted strings: `\\`, `\"`, `\n`, `\r`, `\t`, `\0`, and `\uXXXX`. Both the lowercase `\uXXXX` and the uppercase `\UXXXX` (Apple's canonical form) are accepted.

**Common uses:** ingesting `Localizable.strings` files for iOS/macOS workflows, building configuration profiles, or running diff/merge logic across translation files at plan time.