package cedar

import "testing"

// TestOfficialDocumentsValidate checks that real multi-policy documents from the
// official cedar-policy/cedar-examples repositories parse and format.
func TestOfficialDocumentsValidate(t *testing.T) {
	docs := map[string]string{
		"tinytodo": `permit (
    principal,
    action in [Action::"CreateList", Action::"GetLists"],
    resource == Application::"TinyTodo"
);
permit (principal, action, resource)
when { resource has owner && resource.owner == principal };
permit (
    principal,
    action == Action::"GetList",
    resource
)
when { principal in resource.readers || principal in resource.editors };`,

		"github": `permit (
  principal,
  action == Action::"pull",
  resource
)
when { principal in resource.readers };
permit (
  principal,
  action == Action::"delete_issue",
  resource
)
when { principal in resource.repo.readers && principal == resource.reporter };`,

		"document_cloud": `forbid (
  principal,
  action in [Action::"ViewDocument", Action::"ModifyDocument"],
  resource
)
when
{
  principal has blocked &&
  (resource.owner.blocked.contains(principal) ||
   principal.blocked.contains(resource.owner))
};
forbid (principal, action, resource)
when { !context.is_authenticated };`,

		"streaming_extensions": `@id("rent-buy-oscar-movie")
permit (
  principal is Subscriber,
  action in [Action::"rent", Action::"buy"],
  resource is Movie
)
when
{
  resource.isOscarNominated &&
  context.now.datetime >= datetime("2025-02-02T19:00:00-0500")
};`,
	}
	for name, doc := range docs {
		t.Run(name, func(t *testing.T) {
			if !IsValid(doc) {
				t.Fatalf("official document %q does not validate", name)
			}
			out, err := Format(doc)
			if err != nil {
				t.Fatalf("Format(%q): %v", name, err)
			}
			if !IsValid(out) {
				t.Fatalf("formatted %q is not valid", name)
			}
		})
	}
}

// TestSinglePolicyConstructsRoundTrip exercises operators, extensions, sets,
// records, and the `is`/`like`/`has`/`if-then-else` forms through the EST
// round-trip, using constructs drawn from the official operator reference.
func TestSinglePolicyConstructsRoundTrip(t *testing.T) {
	policies := []string{
		`permit (principal, action, resource) when { principal.age >= 21 };`,
		`permit (principal, action, resource) when { resource is Photo && resource.owner == principal };`,
		`permit (principal, action, resource) when { context.location like "s3:*" };`,
		`permit (principal, action, resource) when { principal has manager && principal.manager == User::"kirk" };`,
		`permit (principal, action, resource) when { [1, 2, 3].contains(1) };`,
		`permit (principal, action, resource) when { ["a", "b"].containsAny(["b", "c"]) };`,
		`permit (principal, action, resource) when { if principal.vip then true else resource.public };`,
		`permit (principal, action, resource) when { ip("10.0.0.1").isInRange(ip("10.0.0.0/24")) };`,
		`permit (principal, action, resource) when { decimal("1.5").greaterThan(decimal("1.0")) };`,
		`permit (principal in Group::"admins", action, resource) unless { resource.locked };`,
	}
	for _, src := range policies {
		want := canonicalSingle(t, src)
		tree, err := Decode(src)
		if err != nil {
			t.Errorf("Decode(%q): %v", src, err)
			continue
		}
		got, err := Encode(tree)
		if err != nil {
			t.Errorf("Encode(Decode(%q)): %v", src, err)
			continue
		}
		if got != want {
			t.Errorf("round-trip mismatch\n  in:   %q\n  want: %q\n  got:  %q", src, want, got)
		}
	}
}
