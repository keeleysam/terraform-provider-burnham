// Derive a full set of UI state colors from one brand color, all in OKLCh.
locals { brand = "#3366cc" }

output "states" {
  value = {
    base     = local.brand
    hover    = provider::burnham::color_adjust(local.brand, { lightness = "*0.92" })
    active   = provider::burnham::color_adjust(local.brand, { lightness = "*0.85" })
    disabled = provider::burnham::color_adjust(local.brand, { chroma = "*0.4", lightness = "+0.1" })
    focus    = provider::burnham::color_adjust(local.brand, { hue = "+20" })
  }
}
