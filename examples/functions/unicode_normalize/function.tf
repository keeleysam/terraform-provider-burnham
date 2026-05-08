/*
Normalise a string to one of the four Unicode normalization forms.

The most common need: `NFC`-normalise input that may have come from a browser, macOS, or a rich-text editor (which often hand you NFD-encoded `é` as `e` + combining acute) so equality comparisons and downstream APIs see consistent bytes.

Caveat: Terraform's value-handling layer (cty) re-normalizes string values to NFC at HCL expression boundaries. NFD and NFKD outputs are silently re-composed to NFC before another HCL expression sees them — those forms are only useful when the function output is consumed before Terraform serialises it.
*/
output "ligature_flattened" {
  value = provider::burnham::unicode_normalize("ﬁne", "NFKC")
  // → "fine"   ("ﬁ" U+FB01 ligature collapses to the two-character "fi" under NFKC)
}

output "noop" {
  value = provider::burnham::unicode_normalize("hello", "NFC")
  // → "hello"
}
