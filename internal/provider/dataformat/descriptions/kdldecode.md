Parses a [KDL document](https://kdl.dev/) string into a Terraform list of node objects.

Each node object has these keys:

- `name` (string)
- `args` (list of values)
- `props` (map of values)
- `children` (list of child nodes)

Both KDL v1 and v2 input are accepted; the parser auto-detects the version.

**Common uses:** reading KDL-based configuration files such as the [`kdl-org/kdl`](https://github.com/kdl-org/kdl) specification documents, Cargo-style nested configuration, or any tool that's adopted KDL as its config format.