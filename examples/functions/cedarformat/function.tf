// cedarformat: canonicalize a hand-written Cedar policy, normalizing layout and
// indentation. Fails on invalid input (use cedarvalidate for a bool check).
output "canonical" {
  value = provider::burnham::cedarformat("permit(principal==User::\"alice\",action==Action::\"view\",resource);")
  /* →
  permit (
      principal == User::"alice",
      action == Action::"view",
      resource
  );
  */
}
