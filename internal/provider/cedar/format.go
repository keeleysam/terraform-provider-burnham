package cedar

import (
	"sort"
	"strconv"
	"strings"

	cedar "github.com/cedar-policy/cedar-go"
)

// IsValid reports whether policies is a syntactically valid Cedar policy document. An empty document (no statements) is valid.
func IsValid(policies string) bool {
	_, err := cedar.NewPolicySetFromBytes("policy.cedar", []byte(policies))
	return err == nil
}

// Format parses a Cedar policy document and returns its canonical DSL serialization: normalized layout and indentation, statements kept in their input order. It errors on syntactically invalid input.
//
// Comments are dropped and formatting is normalized (each policy is re-rendered from the parsed AST). Annotations (@id(...)) are preserved. Each policy is marshaled individually so the output is order-preserving and idempotent (unlike marshaling the whole set, which sorts by policy ID).
func Format(policies string) (string, error) {
	ps, err := cedar.NewPolicySetFromBytes("policy.cedar", []byte(policies))
	if err != nil {
		return "", err
	}
	m := ps.Map()
	// NewPolicySetFromBytes assigns default IDs policy0, policy1, ... by position; sort numerically to recover input order.
	ids := make([]cedar.PolicyID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return policyIndex(ids[i]) < policyIndex(ids[j]) })

	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = string(m[id].MarshalCedar())
	}
	return strings.Join(parts, "\n"), nil
}

// policyIndex extracts the numeric suffix of a default policy ID ("policy0" -> 0), for input-order sorting.
func policyIndex(id cedar.PolicyID) int {
	n, err := strconv.Atoi(strings.TrimPrefix(string(id), "policy"))
	if err != nil {
		return -1
	}
	return n
}
