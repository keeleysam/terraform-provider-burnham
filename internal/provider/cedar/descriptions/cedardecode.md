Parses a single [Cedar](https://www.cedarpolicy.com) policy in its human-readable text form and returns Cedar's JSON policy format, the EST, as a data tree.

It is the inverse of `cedarencode`, so `cedarencode(cedardecode(x))` round-trips to the canonical form of `x`. Use it to inspect, query, or patch a hand-written policy as structured data, or to discover the EST shape to feed back into `cedarencode`.

The return is a dynamic value ready to `jsonencode` into Cedar's JSON policy format.

~> **Note:** It handles exactly one policy statement (the shape of an `aws_verifiedpermissions_policy` static policy). A document with several policies is a policy set and is rejected here; use `cedarformat` or `cedarvalidate` for those.

Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.