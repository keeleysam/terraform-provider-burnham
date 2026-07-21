# PCRE backreferences (which RE2, and Terraform's built-in regex, cannot express):
# does any word immediately repeat itself?
output "has_doubled_word" {
  value = provider::burnham::pcre_match("(\\w+)\\s+\\1\\b", "the the end")
  # → true
}
