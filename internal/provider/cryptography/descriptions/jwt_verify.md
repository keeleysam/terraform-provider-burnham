<!-- Edit here: this is the MarkdownDescription source for the burnham jwt_verify function. docs/functions/jwt_verify.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Verifies a compact [JWS](https://www.rfc-editor.org/rfc/rfc7515) / [JWT](https://www.rfc-editor.org/rfc/rfc7519) signature against `key`, returning an object:

- `valid`: `true` when the signature checks out (and, if `options.now` is supplied, the time claims are within range).
- `header`: the decoded protected header object.
- `payload`: the decoded claims object.

`key` is the shared secret bytes for the HS family, or a PEM `PUBLIC KEY`, `CERTIFICATE`, or private key for ES256 / EdDSA / RS*. The algorithm is read from the token's header.

Time validation is opt-in so the function stays pure by default:

- With no `options.now`, only the signature is checked. Time claims (`exp`, `nbf`) are ignored and the wall clock is never read.
- With `options.now` set (a unix-seconds number or an RFC 3339 string), the token is additionally rejected when `now >= exp` or `now < nbf`. Supply `now` from `plantimestamp()` or a variable to keep the call deterministic.

`options.algorithm` (optional) pins the accepted `alg`: a token whose header algorithm differs is reported `valid = false`. This guards against algorithm-substitution, where an attacker re-signs a token under a different family (for example presenting an `HS256` token to be checked with an RSA public key as the HMAC secret).

-> A malformed token (wrong segment count, bad base64url, non-JSON header or payload) is an error, not `valid = false`. A well-formed token with a bad signature returns `valid = false`.