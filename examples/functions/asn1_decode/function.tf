/*
Generic ASN.1 BER/DER decoder. Walks an arbitrary structure and returns a recursive object tree of {tag, class, compound, type, value, children}. Pair with `pem_decode` to drill into the body of any PEM block.
*/
output "integer_42" {
  value = provider::burnham::asn1_decode("AgEq")
  // → { tag = 2, class = "universal", compound = false, type = "INTEGER", value = "42", children = [] }
}

// Walk a SEQUENCE of two integers.
output "sequence_of_ints" {
  value = [for child in provider::burnham::asn1_decode("MAYCAQECAQI=").children : child.value]
  // → ["1", "2"]
}
