# Reorder with backreferences in the replacement ($1, $2 refer to capture groups).
output "flipped" {
  value = provider::burnham::pcre_replace("(\\w+)@(\\w+)", "user@host", "$2.$1")
  # → "host.user"
}

# Named groups work too. In HCL, escape ${name} as $${name} so Terraform does not
# treat it as interpolation; it reaches the function as the literal ${name}.
output "flipped_named" {
  value = provider::burnham::pcre_replace("(?<user>\\w+)@(?<host>\\w+)", "user@host", "$${host}.$${user}")
  # → "host.user"
}
