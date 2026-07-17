<!-- Edit here: this is the MarkdownDescription source for the burnham dotenvdecode function. docs/functions/dotenvdecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses a [dotenv](https://github.com/joho/godotenv) (`.env`) file body into an object whose attributes are the file's keys. All values are returned as strings, since dotenv has no type system, so cast on the Terraform side with `tonumber()` / `tobool()` when needed.

Parsing rules:

- Comments (`#`) are ignored.
- Both `KEY=value` and `export KEY=value` lines are accepted.
- Double-quoted values support `\n`, `\r` and `${VAR}` interpolation against earlier keys.
- Single-quoted values are taken literally.

Backed by [joho/godotenv](https://github.com/joho/godotenv), the canonical Go implementation.

**Common uses:** ingesting environment files for ECS/Lambda task definitions, container env blocks, or shipping config alongside compiled artifacts.