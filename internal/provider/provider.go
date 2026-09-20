// Package provider implements the Terraform Plugin Framework provider for
// jmnote/wsl. It depends on internal/wsl for every actual interaction with
// wsl.exe; nothing in internal/wsl imports this package or
// terraform-plugin-framework, which is what keeps internal/wsl unit
// testable without Terraform or a real WSL host.
package provider

import (
	"context"
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jmnote/terraform-provider-wsl/internal/wsl"
)

// Ensure WSLProvider satisfies the expected interfaces.
var _ provider.Provider = &WSLProvider{}

// WSLProvider is the jmnote/wsl Terraform provider. It manages the
// lifecycle of WSL instances on the local Windows host that runs
// Terraform; see docs/design/decisions/platform-support.md for the
// execution model this implies and its v0.1.0 scope.
type WSLProvider struct {
	// version is set by main.go via goreleaser at build time (or "dev"
	// for local builds) and reported in provider metadata.
	version string
}

// wslProviderModel is the schema for the `provider "wsl" {}` configuration
// block.
type wslProviderModel struct {
	Executable types.String `tfsdk:"executable"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &WSLProvider{version: version}
	}
}

func (p *WSLProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "wsl"
	resp.Version = p.version
}

func (p *WSLProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the lifecycle of Windows Subsystem for Linux (WSL) instances. " +
			"This provider must run as part of Terraform executing on a Windows host with WSL available; " +
			"it does not manage anything inside an instance (no packages, files, or services).",
		Attributes: map[string]schema.Attribute{
			"executable": schema.StringAttribute{
				Optional: true,
				Description: "Path to, or name on PATH of, the wsl.exe executable to invoke. " +
					"Defaults to \"wsl.exe\", resolved via the standard executable search path.",
			},
		},
	}
}

func (p *WSLProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config wslProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if runtime.GOOS != "windows" {
		resp.Diagnostics.AddError(
			"Unsupported host operating system",
			"The wsl provider drives the wsl.exe command-line tool and only runs as part of Terraform "+
				"executing on Windows. Detected host OS: \""+runtime.GOOS+"\". "+
				"Running Terraform inside a WSL distribution and calling out to /mnt/c/.../wsl.exe is not "+
				"supported in v0.1.0; run Terraform natively on Windows instead.",
		)
		return
	}

	executable := "wsl.exe"
	if !config.Executable.IsNull() && config.Executable.ValueString() != "" {
		executable = config.Executable.ValueString()
	}

	client := wsl.NewClient(wsl.NewProcessRunner(executable))
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *WSLProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
	}
}

func (p *WSLProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewInstanceDataSource,
	}
}
