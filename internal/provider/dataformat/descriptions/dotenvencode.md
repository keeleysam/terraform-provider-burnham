Encodes a flat string-keyed object as `KEY=value` lines in alphabetical key order. Numeric and boolean values are stringified; nested objects and lists are not allowed.

~> **Note:** Keys must be valid POSIX shell identifiers, matching `[A-Za-z_][A-Za-z0-9_]*`. An empty key fails the plan with `dotenv key cannot be empty`; any other invalid key (containing a dot, dash, whitespace, `=`, a quote, etc.) fails with `invalid dotenv key`.

A value is wrapped in double quotes when it contains whitespace, a quote, `$`, `\`, `#`, or a newline. Inside the quotes the encoder escapes so the value round-trips through `dotenvdecode`:

- newline and carriage return become `\n` / `\r`.
- double quote and backslash become `\"` / `\\`.
- `$` becomes `\$`, so `dotenvdecode` does not interpolate `${VAR}` / `$VAR`.
- tab characters are written literally.

**Common uses:** generating `.env` files for `local_file`, container build contexts, or 12-factor service deployments.