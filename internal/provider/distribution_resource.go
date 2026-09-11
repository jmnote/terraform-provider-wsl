package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jmnote/terraform-provider-wsl/internal/wsl"
)

var (
	_ resource.Resource                   = &distributionResource{}
	_ resource.ResourceWithConfigure      = &distributionResource{}
	_ resource.ResourceWithImportState    = &distributionResource{}
	_ resource.ResourceWithValidateConfig = &distributionResource{}
)

func NewDistributionResource() resource.Resource {
	return &distributionResource{}
}

type distributionResource struct {
	client wsl.Client
}

// distributionResourceModel mirrors the wsl_distribution schema. rootfs and
// location are pointers to *the source used at creation time*, not
// necessarily the distribution's current on-disk state -- WSL does not
// expose an API to read either back, so after `terraform import` they
// remain null in state until the practitioner's configuration re-populates
// them (see requiresReplaceUnlessImporting in planmodifiers.go).
type distributionResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Rootfs       types.String `tfsdk:"rootfs"`
	Location     types.String `tfsdk:"location"`
	Distribution types.String `tfsdk:"distribution"`
	Version      types.Int64  `tfsdk:"version"`
	State        types.String `tfsdk:"state"`
}

func (r *distributionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distribution"
}

func (r *distributionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WSL distribution's registration lifecycle, an in-place WSL-version update, " +
			"and delete via `wsl --unregister`. It does not manage anything inside the distribution. " +
			"Creation supports two mutually exclusive modes: bring your own root filesystem tar via " +
			"`rootfs`+`location` (`wsl --import`), or install a Microsoft Store distribution with no tar " +
			"file at all via `distribution` (`wsl --install --distribution`) -- the same primitive everyday, " +
			"interactive `wsl --install <Distribution>` uses. See docs/design-decisions/creation-model.md " +
			"for why both exist and the constraints of each.\n\n" +
			"~> **Destructive delete.** `terraform destroy`, or any change to `name`, `rootfs`, `location`, " +
			"or `distribution`, unregisters the distribution via `wsl --unregister`, which permanently " +
			"deletes its virtual disk and all data inside it. There is no undo and this provider does not " +
			"take backups.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Description: "The distribution's registration name, e.g. \"worker\". This is also the " +
					"identity used by `terraform import`. Changing it requires replacing the resource: WSL " +
					"has no rename operation. In install mode this must equal `distribution`: wsl.exe does " +
					"not support installing a Store distribution under a custom name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rootfs": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Import mode: path to the tar/tar.gz root filesystem archive to import (the " +
					"`wsl --import` source). Required together with `location` when creating an import-mode " +
					"resource; mutually exclusive with `distribution`. WSL exposes no way to read this back " +
					"for an existing distribution, so after `terraform import` this remains unset in state " +
					"until your configuration supplies it.",
				PlanModifiers: []planmodifier.String{
					requiresReplaceUnlessImporting(),
				},
			},
			"location": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Import mode: Windows directory where WSL stores the distribution's virtual " +
					"disk (the `wsl --import` install location). Required together with `rootfs`; mutually " +
					"exclusive with `distribution`. Like `rootfs`, WSL exposes no way to read this back for " +
					"an existing distribution, so after `terraform import` this remains unset in state until " +
					"your configuration supplies it.",
				PlanModifiers: []planmodifier.String{
					requiresReplaceUnlessImporting(),
				},
			},
			"distribution": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Install mode: a Microsoft Store distribution identifier (e.g. " +
					"\"Ubuntu-24.04\"), installed via `wsl --install --distribution` with no tar file needed " +
					"and no other registered distribution touched. Mutually exclusive with `rootfs`/" +
					"`location`. Must equal `name`, since wsl.exe registers it under the Store distribution's " +
					"own name.",
				PlanModifiers: []planmodifier.String{
					requiresReplaceUnlessImporting(),
				},
			},
			"version": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "WSL engine version the distribution runs under: 1 or 2. Changing this performs an in-place `wsl --set-version` rather than replacing the resource, since Microsoft documents that as a supported (if potentially slow) conversion.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Computed: true,
				Description: "The distribution's current run state (e.g. \"Running\"/\"Stopped\") as last " +
					"observed from `wsl --list --verbose`. Informational only: on a non-English Windows " +
					"host this text is localized, so it must not be relied on programmatically.",
			},
		},
	}
}

func (r *distributionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(wsl.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected wsl.Client, got: %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// ValidateConfig catches an invalid creation-mode combination (both modes,
// neither mode, or an incomplete import-mode pair) and an install-mode
// name/distribution mismatch at `terraform plan`/`validate` time, rather
// than only surfacing them as an apply-time error from the wsl.Client
// (which still enforces the same rules as a backstop; see
// internal/wsl/client.go).
func (r *distributionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config distributionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasRootfs := isKnownNonEmpty(config.Rootfs)
	hasLocation := isKnownNonEmpty(config.Location)
	hasDistribution := isKnownNonEmpty(config.Distribution)

	switch {
	case hasDistribution && (hasRootfs || hasLocation):
		resp.Diagnostics.AddError(
			"Conflicting creation mode",
			"distribution is mutually exclusive with rootfs/location: set exactly one creation mode "+
				"(import via rootfs+location, or install via distribution).",
		)
	case hasRootfs != hasLocation:
		resp.Diagnostics.AddError(
			"Incomplete import-mode configuration",
			"rootfs and location must be set together for import mode.",
		)
	case !hasDistribution && !hasRootfs && !hasLocation:
		resp.Diagnostics.AddError(
			"Missing creation mode",
			"either rootfs+location (import mode) or distribution (install mode) must be set.",
		)
	}

	if hasDistribution && isKnownNonEmpty(config.Name) && config.Name.ValueString() != config.Distribution.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"name must equal distribution in install mode",
			"wsl.exe does not support installing a Store distribution under a custom name, so name must "+
				"equal distribution when using install mode.",
		)
	}
}

func isKnownNonEmpty(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

func (r *distributionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan distributionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := wsl.CreateOptions{
		Name:         plan.Name.ValueString(),
		Rootfs:       plan.Rootfs.ValueString(),
		Location:     plan.Location.ValueString(),
		Distribution: plan.Distribution.ValueString(),
	}
	if !plan.Version.IsUnknown() && !plan.Version.IsNull() {
		opts.Version = int(plan.Version.ValueInt64())
	}

	if err := r.client.Create(ctx, opts); err != nil {
		resp.Diagnostics.AddError("Error creating WSL distribution", err.Error())
		return
	}

	resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *distributionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state distributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dist, err := r.client.Get(ctx, state.Name.ValueString())
	if errors.Is(err, wsl.ErrNotFound) {
		// Reconcile drift: something outside Terraform (e.g. `wsl
		// --unregister` run manually) removed this distribution. Removing
		// it from state causes the next plan to propose recreating it,
		// rather than silently drifting or erroring.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading WSL distribution", err.Error())
		return
	}

	state.Version = types.Int64Value(int64(dist.Version))
	state.State = types.StringValue(dist.State)
	// rootfs/location are intentionally left as whatever is already in
	// state: WSL cannot report them back, so Read must not guess at them.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *distributionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state distributionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Version.IsUnknown() && !plan.Version.IsNull() && plan.Version.ValueInt64() != state.Version.ValueInt64() {
		if err := r.client.SetVersion(ctx, state.Name.ValueString(), int(plan.Version.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Error updating WSL distribution version", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *distributionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state distributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting WSL distribution", err.Error())
	}
}

func (r *distributionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The distribution's registration name is the entire import identity;
	// see docs/design-decisions/resource-identity.md. rootfs/location
	// are left null by this and populated by the practitioner's
	// configuration; see requiresReplaceUnlessImporting.
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// refresh re-reads the distribution named in model.Name and overwrites
// model's computed attributes (version, state) with the observed values.
// It is shared by Create and Update so that both always reflect the actual
// post-operation state rather than assuming the requested values took
// effect exactly as asked.
func (r *distributionResource) refresh(ctx context.Context, model *distributionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	dist, err := r.client.Get(ctx, model.Name.ValueString())
	if err != nil {
		diags.AddError("Error reading WSL distribution after apply", err.Error())
		return diags
	}
	model.Version = types.Int64Value(int64(dist.Version))
	model.State = types.StringValue(dist.State)
	return diags
}
