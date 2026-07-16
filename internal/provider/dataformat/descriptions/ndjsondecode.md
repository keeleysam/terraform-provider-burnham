Parses [NDJSON](https://github.com/ndjson/ndjson-spec) (newline-delimited JSON, also called JSON Lines) into a list. Each non-empty line is parsed as an independent JSON value; the result is a tuple containing one element per line.

Blank lines and trailing newlines are tolerated. Numbers preserve precision via `json.Number`.

**Common uses:** ingesting log streams, decoded API event feeds, BigQuery exports, or any line-oriented JSON record format.