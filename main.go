package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/keeleysam/terraform-burnham/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/keeleysam/burnham",
	})
	if err != nil {
		log.Fatal(err)
	}
}
