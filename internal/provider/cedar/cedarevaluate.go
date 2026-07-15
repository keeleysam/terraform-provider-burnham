package cedar

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*CedarEvaluateFunction)(nil)

type CedarEvaluateFunction struct{}

func NewCedarEvaluateFunction() function.Function { return &CedarEvaluateFunction{} }

func (f *CedarEvaluateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedarevaluate"
}

func (f *CedarEvaluateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Evaluate a Cedar authorization request against a policy document",
		MarkdownDescription: "Authorizes a request against a [Cedar](https://www.cedarpolicy.com) policy document and returns the decision, for previewing or unit-testing authorization policies at plan time. Because it uses [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar, the decision comes from Cedar's own evaluation engine rather than an approximation (Amazon Verified Permissions is built on the same engine).\n\nThe second argument is the request object: `principal`, `action`, and `resource` (each an object with `type` and `id`, e.g. `{ type = \"User\", id = \"alice\" }`), an optional `context` (a plain attribute record such as `{ mfa = true }`, referenced in policies as `context.<key>`), and an optional `entities` list. Each entity is `{ uid = { type = ..., id = ... }, attrs = {...}, parents = [ { type = ..., id = ... }, ... ] }`, the Cedar entities shape, providing the attributes and hierarchy the decision resolves against.\n\nReturns an object `{ decision = \"allow\" or \"deny\", reasons = [ids of the policies that determined the decision], errors = [evaluation errors] }`. A policy with no `@id` annotation is numbered `policy0`, `policy1`, and so on in document order, so add `@id(\"...\")` to get stable names in `reasons`.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "policies",
				Description: "A Cedar policy document to authorize against.",
			},
			function.DynamicParameter{
				Name:        "request",
				Description: "The authorization request: principal, action, resource, and optional context and entities.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *CedarEvaluateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var policies string
	var request types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &policies, &request))
	if resp.Error != nil {
		return
	}
	if len(policies) > cedarMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("policy document exceeds maximum supported length of %d bytes", cedarMaxInputBytes))
		return
	}
	if hasUnknown(request) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicUnknown()))
		return
	}

	raw, err := terraformToNode(request.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "failed to read request: "+err.Error())
		return
	}
	evalReq, err := parseRequest(raw)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, err.Error())
		return
	}

	decision, err := Evaluate(policies, evalReq)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	verdict := "deny"
	if decision.Allow {
		verdict = "allow"
	}
	result := map[string]any{
		"decision": verdict,
		"reasons":  toAnyList(decision.Reasons),
		"errors":   toAnyList(decision.Errors),
	}
	value, err := nodeToAttr(result)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}

// parseRequest reads the request object (already lowered to plain Go values) into a Request.
func parseRequest(v any) (Request, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return Request{}, fmt.Errorf("request must be an object")
	}
	for k := range m {
		switch k {
		case "principal", "action", "resource", "context", "entities":
		default:
			return Request{}, fmt.Errorf("unknown request key %q; expected principal, action, resource, context, or entities", k)
		}
	}
	var req Request
	var err error
	if req.Principal, err = entityRef(m["principal"], "principal"); err != nil {
		return Request{}, err
	}
	if req.Action, err = entityRef(m["action"], "action"); err != nil {
		return Request{}, err
	}
	if req.Resource, err = entityRef(m["resource"], "resource"); err != nil {
		return Request{}, err
	}
	if c := m["context"]; c != nil {
		cm, ok := c.(map[string]any)
		if !ok {
			return Request{}, fmt.Errorf("context must be an object")
		}
		req.Context = cm
	}
	if e := m["entities"]; e != nil {
		el, ok := e.([]any)
		if !ok {
			return Request{}, fmt.Errorf("entities must be a list")
		}
		req.Entities = el
	}
	return req, nil
}

func entityRef(v any, what string) (EntityRef, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return EntityRef{}, fmt.Errorf("%s must be an object with type and id", what)
	}
	typ, _ := m["type"].(string)
	id, _ := m["id"].(string)
	if typ == "" || id == "" {
		return EntityRef{}, fmt.Errorf("%s requires a non-empty type and id", what)
	}
	return EntityRef{Type: typ, ID: id}, nil
}

func toAnyList(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
