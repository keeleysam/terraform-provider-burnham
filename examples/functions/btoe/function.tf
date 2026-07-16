// btoe (RFC 1751 "bytes to english"): render a key as human-readable words.
// Each 64-bit block becomes six words; a 128-bit key yields twelve.
output "key_words" {
  value = provider::burnham::btoe("CCAC2AED591056BE4F90FD441C534766")
  // → "RASH BUSH MILK LOOK BAD BRIM AVID GAFF BAIT ROT POD LOVE"
}
