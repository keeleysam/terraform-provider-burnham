// celdecode: parse a CEL string into the celencode data tree. Useful for
// testing (round-tripping) and for migrating hand-written CEL into the model.
output "roundtrip" {
  value = provider::burnham::celencode(provider::burnham::celdecode(
    "device.os_type == OsType.DESKTOP_MAC && origin.region_code in ['US', 'CA']"
  ))
  // → device.os_type == OsType.DESKTOP_MAC && origin.region_code in ["US", "CA"]
}

// Choose the returned notation: "canonical", "standard" (default), or "aliased".
output "aliased_tree" {
  value = provider::burnham::celdecode("a == 1 && b != 2", { notation = "aliased" })
  // → { and = [ { eq = [{ ident = "a" }, 1] }, { ne = [{ ident = "b" }, 2] } ] }
}
