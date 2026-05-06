// Build a .reg file. Use regdword / regbinary / regmulti / etc. for typed values.
output "registry_export" {
  value = provider::burnham::regencode({
    "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
      DisplayName = "My Application"
      Version     = provider::burnham::regdword(66051)
    }
  })
}
/* →
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp]
"DisplayName"="My Application"
"Version"=dword:00010203
*/
