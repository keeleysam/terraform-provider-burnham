# Typeset an invoice from structured data. The `inputs` object is exposed to the
# document as sys.inputs (as native Typst values), so the source reads it directly.
locals {
  invoice = <<-TYPST
    #set text(font: "Noto Sans")
    #set page(width: 320pt, height: auto, margin: 24pt)
    = Invoice #sys.inputs.number
    Bill to: *#sys.inputs.customer*
    #line(length: 100%)
    Total due: #sys.inputs.total
  TYPST
}

output "invoice_pdf" {
  value = provider::burnham::typst_pdf(local.invoice, {
    inputs = {
      number   = "INV-2026-014"
      customer = "Ada Lovelace"
      total    = "$1,240.00"
    }
  })
  # → a base64-encoded PDF (write it with a local_file's content_base64, or decode with base64decode)
}
