# Render a one-page badge to PNG at 220 ppi. typst_png returns a list with one
# base64 PNG per page, so a single-page document is result[0].
locals {
  badge = <<-TYPST
    #set page(width: 180pt, height: 100pt, margin: 0pt, fill: rgb("#1d4ed8"))
    #set text(font: "Noto Sans", fill: white)
    #align(center + horizon)[
      #text(size: 20pt, weight: "bold")[#sys.inputs.name] \
      #text(size: 11pt)[Level #sys.inputs.level]
    ]
  TYPST
}

output "badge_png" {
  value = provider::burnham::typst_png(local.badge, {
    inputs = { name = "Ada", level = 7 }
    ppi    = 220
  })[0]
  # → a base64-encoded PNG of the badge
}
