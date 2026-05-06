# Build a .reg file. Use regdword / regbinary / regmulti / etc. for typed values.
output "registry_export" {
  value = provider::burnham::regencode({
    "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
      DisplayName = "My Application"
      Version     = provider::burnham::regdword(0x10203)
    }
  })
}
