# Export a document to a single self-contained HTML string (experimental target):
# CSS is inlined and any images embed as data URLs, so there are no external files.
locals {
  page = <<-TYPST
    = #sys.inputs.title
    A short, semantic document with a list:
    - first
    - second
    and a #link("https://typst.app")[link].
  TYPST
}

output "page_html" {
  value = provider::burnham::typst_html(local.page, {
    inputs = { title = "Release notes" }
  })
  # → a self-contained HTML string
}
