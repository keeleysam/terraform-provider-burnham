Parses an Apple [`.strings`](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPInternational/MaintaingYourOwnStringsFiles/MaintaingYourOwnStringsFiles.html) localization file body into a flat string-to-string object. Both UTF-8 and UTF-16 (with BOM) inputs are auto-detected. `//` and `/* */` comments are tolerated and skipped.

Each entry follows `"key" = "value";` with C-style escapes inside the quoted strings: `\\`, `\"`, `\n`, `\r`, `\t`, and `\uXXXX`.

**Common uses:** ingesting `Localizable.strings` files for iOS/macOS workflows, building configuration profiles, or running diff/merge logic across translation files at plan time.