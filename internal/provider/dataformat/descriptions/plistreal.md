Returns a tagged object representing a plist `<real>` (floating-point) value. `plistencode` would normally encode whole numbers as `<integer>` and only fractional numbers as `<real>`, so this helper forces a whole number into `<real>` form when the consumer expects a floating-point type. `plistdecode` returns the same tagged-object shape for whole-number `<real>` elements, preserving the type across round-trips.

Fractional values like `3.14` already encode as `<real>` automatically, so this helper is only needed for whole-number reals.

**Common uses:** profile fields that demand a floating-point type even when the value happens to be a whole number (e.g. some `Rating`, `Score`, or version fields in MDM payloads).