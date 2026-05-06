// Tagged REG_EXPAND_SZ (string with %VAR% expansion) for use inside regencode.
output "default_app_data" {
  value = provider::burnham::regexpandsz("%APPDATA%\\MyApp")
  // → { __reg_type = "expand_sz", value = "%APPDATA%\\MyApp" }
}
