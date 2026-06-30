// dedent — remove the common leading whitespace from every line (textwrap.dedent).
// Handy for indented config/scripts pulled from a variable or a plain (non-indented) heredoc.
output "script" {
  value = provider::burnham::dedent("    if x:\n        y")
  // → "if x:\n    y"
}
