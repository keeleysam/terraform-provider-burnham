package cedar

import (
	"bytes"
	"encoding/json"
	"fmt"

	cedar "github.com/cedar-policy/cedar-go"
)

// Decode parses a single Cedar policy in the human-readable DSL syntax and returns its EST (the Cedar JSON policy format) as a data tree, the inverse of Encode. cedarencode(cedardecode(x)) round-trips to the canonical form of x.
//
// It handles exactly one policy statement (the shape of an AWS Verified Permissions static policy); a document with several policies is a policy set and is rejected here (use Format/IsValid/Evaluate for those). Templates (?principal/?resource) are not static policies and fail to parse.
func Decode(policy string) (any, error) {
	if err := checkNestingDepth(policy); err != nil {
		return nil, err
	}
	ps, err := cedar.NewPolicySetFromBytes("policy.cedar", []byte(policy))
	if err != nil {
		return nil, err
	}
	m := ps.Map()
	if len(m) != 1 {
		return nil, fmt.Errorf("cedardecode handles a single policy, but the input has %d; use cedarformat or cedarvalidate for a multi-policy document", len(m))
	}

	var p *cedar.Policy
	for _, pol := range m {
		p = pol
	}
	j, err := p.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("encode policy as EST JSON: %w", err)
	}

	// UseNumber keeps integer literals exact: Cedar's Long is a 64-bit integer, and the default decoder would widen every number to float64 and lose precision beyond 2^53.
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.UseNumber()
	var tree any
	if err := dec.Decode(&tree); err != nil {
		return nil, fmt.Errorf("decode EST JSON: %w", err)
	}
	return tree, nil
}
