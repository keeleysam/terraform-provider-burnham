Encodes a flat string-keyed object as an Apple `.strings` localization file body. Output is UTF-8 with `"key" = "value";` lines in alphabetical key order.

Inside the quoted strings these characters are escaped, and other control characters pass through unchanged:

- backslash becomes `\\`.
- double quote becomes `\"`.
- newline, carriage return, and tab become `\n`, `\r`, `\t`.

Nested objects and lists are not allowed.

-> **Note:** Modern Xcode toolchains (Xcode 13+) accept UTF-8 `.strings` files. Older tooling may require UTF-16 conversion, which you can do with `iconv` after writing the file via `local_file`.