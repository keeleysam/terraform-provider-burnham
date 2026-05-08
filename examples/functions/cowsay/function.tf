// Render a message inside the ASCII speech bubble of a cow. The classic Unix cowsay(1) layout, self-contained — no external `cowsay` binary involved.
output "default" {
  value = provider::burnham::cowsay("Hello, world.")
}

// "think" mode swaps the bubble brackets to ( ) and uses 'o' connectors.
output "thinking" {
  value = provider::burnham::cowsay("hmm", { action = "think" })
}

// Custom eyes (must be exactly 2 characters). Common alternatives: "==" (drowsy), "@@" (paranoid), "--" (dead).
output "drowsy" {
  value = provider::burnham::cowsay("Whoa.", { eyes = "==" })
}
