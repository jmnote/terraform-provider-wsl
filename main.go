// Terraform Plugin Framework entry point for the jmnote/wsl provider.
//
// Run `go generate ./...` to refresh the generated documentation under
// docs/ after changing any schema (requires
// github.com/hashicorp/terraform-plugin-docs' tfplugindocs; see README.md).
//
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/jmnote/terraform-provider-wsl/internal/provider"
)

// version is overwritten at build time by GoReleaser via -ldflags, e.g.
// -X main.version=1.2.3. It is left at "dev" for local `go build`/`go run`.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Address matches the Terraform Registry source used in
		// `required_providers`: registry.terraform.io/jmnote/wsl.
		Address: "registry.terraform.io/jmnote/wsl",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
