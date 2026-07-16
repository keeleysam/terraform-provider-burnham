Base32-encodes the input's bytes per [RFC 4648](https://www.rfc-editor.org/rfc/rfc4648). Terraform core has no base32 function; with no options this produces standard, padded base32. The optional object selects the variant:

- `hex_alphabet` (bool, default `false`): use the extended-hex alphabet (`0–9A–V`, §7) instead of the standard one (`A–Z2–7`, §6). The hex alphabet sorts in the same order as the underlying bytes and is used by DNSSEC NSEC3.
- `padding` (bool, default `true`): emit `=` padding. Set `false` for the raw form (e.g. TOTP/MFA secrets are unpadded standard base32).

The input is taken as raw bytes (the literal UTF-8 bytes of the string); to encode bytes held as hex, pass `hexdecode(var.x)`.

```
base32encode("foobar")                  → "MZXW6YTBOI======"
base32encode(var.secret, { padding = false })
```