Parses [NDJSON](https://github.com/ndjson/ndjson-spec) (newline-delimited JSON, also called JSON Lines) into a list. Parsing follows the JSON grammar (a streaming decoder), not line boundaries, so each JSON value in the stream becomes one tuple element. Standard NDJSON (one value per line) behaves as expected.

Blank lines and trailing newlines are tolerated. Numbers preserve precision via `json.Number`.

**Common uses:** ingesting log streams, decoded API event feeds, BigQuery exports, or any line-oriented JSON record format.