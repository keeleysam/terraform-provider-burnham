# Render an object as Valve VDF.
output "appstate" {
  value = provider::burnham::vdfencode({
    AppState = { appid = "730", name = "Counter-Strike 2" }
  })
}
