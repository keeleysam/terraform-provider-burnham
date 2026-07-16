Parses an INI string into a Terraform object. The result is a two-level map: section names at the outer level, key/value pairs at the inner level. Keys outside any section ("global" keys) are placed under the empty-string key (`""`) so the structure stays uniform.

All values are strings, since INI has no native type system, so cast on the Terraform side with `tonumber()` / `tobool()` when needed.

**Common uses:** reading legacy application config (`my.cnf`, `php.ini`, `.gitconfig`-style files), normalizing operator-edited config into a typed Terraform value, or feeding INI content into a `templatefile` substitution.