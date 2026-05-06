// Tagged REG_MULTI_SZ (list of strings) for use inside a regencode payload.
output "search_paths" {
  value = provider::burnham::regmulti(["C:\\bin", "C:\\tools"])
  // → { __reg_type = "multi_sz", value = ["C:\\bin", "C:\\tools"] }
}
