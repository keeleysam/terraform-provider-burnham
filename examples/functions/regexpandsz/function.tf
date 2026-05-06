# Tagged REG_EXPAND_SZ (string with %VAR% expansion) for regencode.
output "default_app_data" {
  value = provider::burnham::regexpandsz("%APPDATA%\\MyApp")
}
