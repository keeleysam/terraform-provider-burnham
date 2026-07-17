// Pick legible label text for each team's background color.
variable "team_colors" {
  type    = map(string)
  default = { platform = "#1e3a8a", design = "#fde047", sre = "#7f1d1d" }
}

output "label_text_colors" {
  value = { for team, bg in var.team_colors : team => provider::burnham::color_readable_text(bg) }
  // → { platform = "#ffffff", design = "#000000", sre = "#ffffff" }
}
