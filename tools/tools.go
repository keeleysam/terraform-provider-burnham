//go:build tools

// Package tools tracks build-time-only dependencies. The blank import keeps
// `go mod tidy` from removing them while the build tag prevents them from
// being compiled into the production binary.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
