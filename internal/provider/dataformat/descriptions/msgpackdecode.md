Decodes [MessagePack](https://msgpack.org/) bytes into a Terraform value. Provide the bytes as a standard base64 string, since HCL strings are UTF-8 only.

Type mapping:

- maps become objects
- arrays become tuples
- integers and floats become numbers
- binary blobs (msgpack `bin` format) become base64 strings

~> **Note:** Extension types are not supported.

**Common uses:** consuming msgpack-encoded payloads from caches (Redis, etcd), inspecting `kubectl get --raw` output, or round-tripping fixtures.