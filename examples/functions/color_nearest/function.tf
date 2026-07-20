// Snap an arbitrary color onto a fixed brand palette. The closest entry is
// returned exactly as written, so the result stays in your canonical spelling.
locals {
  brand_palette = ["#1e3a8a", "#b91c1c", "#15803d", "#a16207"]
}

output "snapped" {
  value = provider::burnham::color_nearest("#1d40ab", local.brand_palette)
  // → "#1e3a8a"
}

// Quantize a computed color to the 8 basic named colors (returned by name).
output "quantized" {
  value = provider::burnham::color_nearest(
    "#2f6fdb",
    ["black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"],
  )
  // → "blue"
}
