package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is the standard terraform-plugin-testing
// wiring for acceptance tests: it serves this provider in-process so
// resource.Test can drive a real Terraform apply/destroy cycle against it.
//
// These acceptance tests are gated by TF_ACC=1 (resource.Test skips
// otherwise) and additionally require a real Windows host with WSL
// installed, since -- unlike internal/wsl's unit tests -- they exercise
// the actual provider binary talking to the actual wsl.exe. See
// README.md, "Acceptance tests".
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"wsl": providerserver.NewProtocol6WithError(New("acctest")()),
}
