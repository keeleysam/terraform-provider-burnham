package cedar

import (
	"encoding/json"
	"fmt"

	cedar "github.com/cedar-policy/cedar-go"
	"github.com/cedar-policy/cedar-go/types"
)

// EntityRef identifies a Cedar entity by type and id, e.g. {Type: "User", ID: "alice"} for User::"alice".
type EntityRef struct {
	Type string
	ID   string
}

// Request is an authorization request evaluated against a policy set: who (Principal) is trying to do what (Action) to what (Resource), plus the request Context and the Entities store the decision resolves against.
type Request struct {
	Principal EntityRef
	Action    EntityRef
	Resource  EntityRef
	Context   map[string]any // the Cedar context record (JSON-shaped)
	Entities  []any          // the Cedar entities JSON array: [{uid, attrs, parents}, ...]
}

// Decision is the result of Evaluate: whether the request is allowed, the IDs of the policies that determined it, and any evaluation errors.
type Decision struct {
	Allow   bool
	Reasons []string
	Errors  []string
}

// Evaluate authorizes req against the given Cedar policy document and returns the decision. Because it uses cedar-go, the official implementation, the decision matches the engine Amazon Verified Permissions runs.
func Evaluate(policies string, req Request) (Decision, error) {
	if err := checkNestingDepth(policies); err != nil {
		return Decision{}, err
	}
	ps, err := cedar.NewPolicySetFromBytes("policy.cedar", []byte(policies))
	if err != nil {
		return Decision{}, err
	}

	entities := types.EntityMap{}
	if len(req.Entities) > 0 {
		b, err := json.Marshal(req.Entities)
		if err != nil {
			return Decision{}, fmt.Errorf("entities: %w", err)
		}
		if err := json.Unmarshal(b, &entities); err != nil {
			return Decision{}, fmt.Errorf("entities: %w", err)
		}
	}

	var ctx types.Record
	if len(req.Context) > 0 {
		b, err := json.Marshal(req.Context)
		if err != nil {
			return Decision{}, fmt.Errorf("context: %w", err)
		}
		if err := json.Unmarshal(b, &ctx); err != nil {
			return Decision{}, fmt.Errorf("context: %w", err)
		}
	}

	cedarReq := cedar.Request{
		Principal: types.NewEntityUID(types.EntityType(req.Principal.Type), types.String(req.Principal.ID)),
		Action:    types.NewEntityUID(types.EntityType(req.Action.Type), types.String(req.Action.ID)),
		Resource:  types.NewEntityUID(types.EntityType(req.Resource.Type), types.String(req.Resource.ID)),
		Context:   ctx,
	}

	decision, diag := cedar.Authorize(ps, entities, cedarReq)

	out := Decision{Allow: decision == cedar.Allow}
	for _, r := range diag.Reasons {
		out.Reasons = append(out.Reasons, string(r.PolicyID))
	}
	for _, e := range diag.Errors {
		out.Errors = append(out.Errors, fmt.Sprintf("%v", e))
	}
	return out, nil
}
