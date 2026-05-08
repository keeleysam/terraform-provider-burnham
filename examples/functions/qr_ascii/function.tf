/*
Render a QR code as Unicode-block ASCII art. Useful for embedding scannable codes in cloud-init MOTDs, login banners, or PDF source — anywhere a text representation beats an image asset.

Each terminal row encodes two QR module rows via the half-block characters ▀ ▄ █, so the output is roughly half as tall as a one-cell-per-module renderer.
*/
output "url" {
  value = provider::burnham::qr_ascii("https://terraform.io")
}

// Higher error correction → more redundancy, larger code, survives more occlusion.
output "high_ec" {
  value = provider::burnham::qr_ascii("https://terraform.io", { error_correction = "H" })
}

// Inverted style for black-background terminals.
output "light_on_dark" {
  value = provider::burnham::qr_ascii("https://terraform.io", { style = "light_on_dark" })
}
