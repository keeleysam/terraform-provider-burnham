terraform {
  required_providers {
    burnham = {
      source = "keeleysam/burnham"
    }
  }
}

# Burnham functions run at plan time and take no provider configuration.
# They assemble correct artifacts from Terraform data, filling gaps the
# built-in expression language cannot cover cleanly on its own.

locals {
  # Pretty, human-diffable JSON. Terraform's built-in jsonencode emits a single line.
  app_config = provider::burnham::jsonencode(
    {
      name     = "checkout"
      replicas = 3
      env      = { LOG_LEVEL = "info" }
    },
    { indent = "  " },
  )

  # Real CIDR set arithmetic instead of a templatefile()-driven preprocessor:
  # the first free /24 in 10.0.0.0/16, given what is already allocated.
  next_subnet = provider::burnham::cidr_find_free(
    ["10.0.0.0/16"],
    ["10.0.0.0/24", "10.0.1.0/24"],
    24,
  )

  # A stable, deterministic identifier that never churns the plan.
  service_id = provider::burnham::uuid_v5("dns", "checkout.example.com")
}
