// Derive a hover shade by mixing the brand color 15% toward black (in OKLCh).
output "button_hover" {
  value = provider::burnham::color_mix("#3366cc", "#000000", 0.15)
}
