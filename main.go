package main

// Order matters: gendoctemplates writes per-function templates under templates/functions/ (gitignored) with the right `subcategory:` baked in based on which package's Functions() registers each function. tfplugindocs then reads those templates to render docs/functions/.
//go:generate go run ./cmd/gendoctemplates
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name burnham

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/keeleysam/terraform-burnham/internal/provider"
)

func main() {
	// `--debug` lets a debugger (delve, IDE-attached debuggers, etc.) attach to the running provider. terraform-plugin-framework's providerserver prints a TF_REATTACH_PROVIDERS line on startup that the editor's Terraform integration can pick up.
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/keeleysam/burnham",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
