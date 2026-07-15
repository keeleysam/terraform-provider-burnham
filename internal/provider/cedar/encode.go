// Package cedar provides Terraform provider functions to encode, decode, validate, format, and evaluate Cedar authorization policies.
//
// Cedar (https://www.cedarpolicy.com) is the policy language behind Amazon Verified Permissions and AWS IAM Access Analyzer. A policy has a human-readable DSL form (permit/forbid statements) and an equivalent canonical JSON form, the EST. These functions convert between the two, check and canonicalize the DSL, and evaluate authorization requests.
//
// Backed by github.com/cedar-policy/cedar-go, the official Go implementation, so a locally computed authorization decision matches the engine Amazon Verified Permissions runs.
package cedar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	cedar "github.com/cedar-policy/cedar-go"
)

// errInvalidOutput signals that the EST data tree did not describe a valid Cedar policy, so no DSL could be built from it. It lets callers attribute the failure.
var errInvalidOutput = errors.New("EST data tree is not a valid Cedar policy")

// Encode builds a single Cedar policy in the DSL syntax from an EST data tree (the Cedar JSON policy format), the inverse of Decode.
//
// The tree is validated as it is converted, so Encode never emits a syntactically invalid policy, and the output is canonical (byte-identical to what Format produces for one policy).
func Encode(tree any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// Cedar EST keys include operators like "&&"; keep them literal rather than &-escaped (cedar-go accepts either, but the unescaped JSON is cleaner).
	enc.SetEscapeHTML(false)
	if err := enc.Encode(tree); err != nil {
		return "", fmt.Errorf("encode EST as JSON: %w", err)
	}
	var p cedar.Policy
	if err := p.UnmarshalJSON(buf.Bytes()); err != nil {
		return "", fmt.Errorf("%w: %v", errInvalidOutput, err)
	}
	return string(p.MarshalCedar()), nil
}
