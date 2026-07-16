/*
base64brotli: Brotli compression (RFC 7932) then base64.

~8–10% smaller than base64gzip on text-heavy payloads, at the cost of a brotli decompressor on the consuming side. Decompress with `base64 -d | brotli -d`. The encoder is deterministic, so plans stay stable.
*/

# Common case: text payload, maximum ratio, default 4 MiB window.
output "boot_scripts_blob" {
  value = provider::burnham::base64brotli(jsonencode({ install = "...", configure = "..." }))
}

# Lower quality for faster plans on very large synthesized inputs.
output "fast" {
  value = provider::burnham::base64brotli(jsonencode({ install = "...", configure = "..." }), { quality = 6 })
}
