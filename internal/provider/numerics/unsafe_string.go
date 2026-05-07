package numerics

import "unsafe"

// bytesToStringNoCopy converts a freshly-allocated []byte to a string
// without the copy that string([]byte) would induce. This is the same
// trick strings.Builder.String() uses internally.
//
// Caller obligations:
//   - the input slice was allocated for this purpose and is not aliased,
//   - the input slice is not read or written after the conversion.
//
// Violating either makes the result observe mutations to the underlying
// memory, which the language model otherwise forbids for strings.
func bytesToStringNoCopy(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
