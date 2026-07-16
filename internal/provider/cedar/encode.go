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
	"sort"

	cedarast "github.com/cedar-policy/cedar-go/ast"
	xast "github.com/cedar-policy/cedar-go/x/exp/ast"
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
	var p cedarast.Policy
	if err := p.UnmarshalJSON(buf.Bytes()); err != nil {
		return "", fmt.Errorf("%w: %v", errInvalidOutput, err)
	}
	// cedar-go builds a record node's element slice by iterating a Go map (see recordJSON.ToNode), so EST JSON -> AST leaves record-literal keys in random order and MarshalCedar would emit them differently on every call. EST records are JSON objects with no inherent order, so sort keys into one canonical order to keep this plan-time function deterministic (otherwise Terraform sees perpetual diffs).
	sortRecordKeys(&p)
	return string(p.MarshalCedar()), nil
}

// sortRecordKeys rewrites every record literal in the policy's conditions so its keys are in lexicographic order, in place. Inspect hands back each node by value, but the record's Elements slice shares its backing array with the AST, so sorting through that header reorders the real node.
func sortRecordKeys(p *cedarast.Policy) {
	for _, cond := range (*xast.Policy)(p).Conditions {
		xast.Inspect(xast.NewNode(cond.Body), func(n xast.IsNode) bool {
			if rec, ok := n.(xast.NodeTypeRecord); ok {
				sort.Slice(rec.Elements, func(i, j int) bool {
					return rec.Elements[i].Key < rec.Elements[j].Key
				})
			}
			return true
		})
	}
}
