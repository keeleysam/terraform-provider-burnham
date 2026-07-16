// celformat: canonicalize (and optionally pretty-print) a hand-written CEL string.
// Fails the plan if the expression is not syntactically valid.
output "canonical" {
  value = provider::burnham::celformat("request.time < timestamp('2026-01-01T00:00:00Z') && 'admin' in user.roles")
  // → request.time < timestamp("2026-01-01T00:00:00Z") && "admin" in user.roles
}

// Reformat with cel-go / celfmt options: wrap at a column, choose operators, newline placement.
output "wrapped" {
  value = provider::burnham::celformat(
    "aaaaa == 1 && bbbbb == 2 && ccccc == 3 && ddddd == 4",
    { format = { wrap_on_column = 30, wrap_on_operators = ["&&"], wrap_after_column_limit = true } },
  )
  /* →
     aaaaa == 1 && bbbbb == 2 && ccccc == 3 &&
     ddddd == 4
  */
}
