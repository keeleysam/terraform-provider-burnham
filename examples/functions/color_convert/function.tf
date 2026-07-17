// Normalize one brand color for two providers with different hex conventions.
output "github_label_color" {
  // github_issue_label wants six hex digits with no leading "#".
  value = provider::burnham::color_convert("rebeccapurple", "hex", { hash = false })
  // → "663399"
}

output "gitlab_label_color" {
  // gitlab_label accepts a leading "#".
  value = provider::burnham::color_convert("rgb(102, 51, 153)", "hex")
  // → "#663399"
}
