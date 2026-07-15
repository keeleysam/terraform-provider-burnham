package cedar

import (
	"reflect"
	"testing"
)

func TestEvaluate(t *testing.T) {
	// The cedar-go README quick-start: alice may view a photo in an album she can access.
	readmePolicy := `permit (principal == User::"alice", action == Action::"view", resource in Album::"jane_vacation");`
	readmeEntities := []any{
		map[string]any{"uid": map[string]any{"type": "User", "id": "alice"}, "attrs": map[string]any{"age": 18}, "parents": []any{}},
		map[string]any{"uid": map[string]any{"type": "Photo", "id": "vacay.jpg"}, "attrs": map[string]any{}, "parents": []any{map[string]any{"type": "Album", "id": "jane_vacation"}}},
	}

	cases := []struct {
		name       string
		policies   string
		req        Request
		wantAllow  bool
		wantReason []string
	}{
		{
			"allow: alice views photo in her album",
			readmePolicy,
			Request{
				Principal: EntityRef{"User", "alice"}, Action: EntityRef{"Action", "view"}, Resource: EntityRef{"Photo", "vacay.jpg"},
				Entities: readmeEntities,
			},
			true, []string{"policy0"},
		},
		{
			"deny: bob is not alice",
			readmePolicy,
			Request{
				Principal: EntityRef{"User", "bob"}, Action: EntityRef{"Action", "view"}, Resource: EntityRef{"Photo", "vacay.jpg"},
				Entities: readmeEntities,
			},
			false, nil,
		},
		{
			"forbid overrides permit on a private resource",
			`permit (principal, action, resource);
forbid (principal, action, resource) when { resource.private };`,
			Request{
				Principal: EntityRef{"User", "alice"}, Action: EntityRef{"Action", "view"}, Resource: EntityRef{"Doc", "secret"},
				Entities: []any{map[string]any{"uid": map[string]any{"type": "Doc", "id": "secret"}, "attrs": map[string]any{"private": true}, "parents": []any{}}},
			},
			false, []string{"policy1"},
		},
		{
			"context gates the decision (mfa present)",
			`permit (principal, action, resource) when { context.mfa };`,
			Request{
				Principal: EntityRef{"User", "alice"}, Action: EntityRef{"Action", "view"}, Resource: EntityRef{"Doc", "d1"},
				Context: map[string]any{"mfa": true},
			},
			true, []string{"policy0"},
		},
		{
			"context gates the decision (mfa absent)",
			`permit (principal, action, resource) when { context.mfa };`,
			Request{
				Principal: EntityRef{"User", "alice"}, Action: EntityRef{"Action", "view"}, Resource: EntityRef{"Doc", "d1"},
				Context: map[string]any{"mfa": false},
			},
			false, nil,
		},
		{
			"group hierarchy via parents",
			`permit (principal in Group::"admins", action, resource);`,
			Request{
				Principal: EntityRef{"User", "alice"}, Action: EntityRef{"Action", "view"}, Resource: EntityRef{"Doc", "d1"},
				Entities: []any{map[string]any{"uid": map[string]any{"type": "User", "id": "alice"}, "attrs": map[string]any{}, "parents": []any{map[string]any{"type": "Group", "id": "admins"}}}},
			},
			true, []string{"policy0"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate(tc.policies, tc.req)
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}
			if got.Allow != tc.wantAllow {
				t.Fatalf("Allow = %v, want %v (reasons %v, errors %v)", got.Allow, tc.wantAllow, got.Reasons, got.Errors)
			}
			if len(got.Errors) != 0 {
				t.Fatalf("unexpected evaluation errors: %v", got.Errors)
			}
			if !reflect.DeepEqual(got.Reasons, tc.wantReason) {
				t.Fatalf("Reasons = %v, want %v", got.Reasons, tc.wantReason)
			}
		})
	}
}

func TestEvaluateInvalidPolicy(t *testing.T) {
	_, err := Evaluate(`permit (`, Request{Principal: EntityRef{"User", "a"}, Action: EntityRef{"Action", "x"}, Resource: EntityRef{"R", "r"}})
	if err == nil {
		t.Fatal("Evaluate with invalid policy should error")
	}
}
