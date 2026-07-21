# A two-page document rendered to SVG. typst_svg returns one SVG string per page,
# as text (not base64), so you can use it directly or feed it to svg_render.
locals {
  report = <<-TYPST
    #set text(font: "Noto Serif")
    #set page(width: 240pt, height: 160pt, margin: 20pt)
    = Page one
    Summary for #sys.inputs.team.
    #pagebreak()
    = Page two
    Details follow.
  TYPST
}

output "report_svgs" {
  value = provider::burnham::typst_svg(local.report, {
    inputs = { team = "Platform" }
  })
  # → a list of two SVG strings, one per page
}
