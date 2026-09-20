package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jmnote/terraform-provider-wsl/internal/wsl"
)

var (
	_ datasource.DataSource              = &instanceDataSource{}
	_ datasource.DataSourceWithConfigure = &instanceDataSource{}
)

func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

type instanceDataSource struct {
	client wsl.Client
}

type instanceDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	Version types.Int64  `tfsdk:"version"`
	State   types.String `tfsdk:"state"`
}

func (d *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing WSL instance registered on the host.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The instance's registration name.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "WSL engine version the instance runs under: 1 or 2.",
			},
			"state": schema.StringAttribute{
				Computed: true,
				Description: "The instance's current run state (e.g. \"Running\"/\"Stopped\") as " +
					"observed from `wsl --list --verbose`. Informational only: on a non-English Windows " +
					"host this text is localized.",
			},
		},
	}
}

func (d *instanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(wsl.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			fmt.Sprintf("Expected wsl.Client, got: %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config instanceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dist, err := d.client.Get(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading WSL instance", err.Error())
		return
	}

	config.Version = types.Int64Value(int64(dist.Version))
	config.State = types.StringValue(dist.State)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
