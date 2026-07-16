/*
base64zopfli: a tighter, drop-in replacement for the built-in base64gzip.

Output is an ordinary RFC 1952 gzip member, so the consumer side decompresses with `gunzip` exactly as before, only the bytes are a few percent smaller. Same input always produces byte-identical output (MTIME is pinned to 0), so plans stay stable.
*/

# Drop-in swap for base64gzip, nothing else in the pipeline changes.
output "boot_scripts_blob" {
  value = provider::burnham::base64zopfli(jsonencode({ install = "...", configure = "..." }))
}

# Spend more CPU at plan time for a slightly smaller result.
output "tighter" {
  value = provider::burnham::base64zopfli(jsonencode({ install = "...", configure = "..." }), { iterations = 100 })
}
