# Stateless ULA → public-prefix translation, checksum-neutral.
# Reverse a translation by swapping the from/to prefixes.
output "external_address" {
  value = provider::burnham::nptv6_translate(
    "fd00::10:0:1",
    "fd00::/48",      # internal
    "2001:db8::/48",  # external
  )
}
