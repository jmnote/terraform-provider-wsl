package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// requiresReplaceUnlessImporting behaves like
// stringplanmodifier.RequiresReplace(), except it does not force a replace
// when the prior state value is null. That null state only occurs right
// after `terraform import`: WSL exposes no way to recover a distribution's
// original rootfs archive path or install location (see
// docs/design-decisions/observable-state.md), so those attributes are
// left unset in state by ImportState.
// Without this modifier, the first `terraform plan` a user runs after
// import -- once they write matching rootfs/location values into
// configuration, which they must do to manage the resource going forward --
// would see state(null) != config(value) on a RequiresReplace attribute and
// plan to destroy and recreate the distribution it was supposed to be
// adopting. This modifier lets that first plan simply adopt the configured
// value into state instead. Genuine changes to an already-known value still
// require replacement, exactly as stringplanmodifier.RequiresReplace does.
func requiresReplaceUnlessImporting() planmodifier.String {
	return requiresReplaceUnlessImportingModifier{}
}

type requiresReplaceUnlessImportingModifier struct{}

func (m requiresReplaceUnlessImportingModifier) Description(ctx context.Context) string {
	return m.MarkdownDescription(ctx)
}

func (m requiresReplaceUnlessImportingModifier) MarkdownDescription(_ context.Context) string {
	return "Requires replacement if the value changes, except immediately after `terraform import`, " +
		"when the prior value is unknown and the configured value is instead adopted into state."
}

func (m requiresReplaceUnlessImportingModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() {
		// Freshly imported: nothing to compare against, so adopt the
		// configured value without flagging a replace.
		return
	}

	delegate := stringplanmodifier.RequiresReplace()
	delegate.PlanModifyString(ctx, req, resp)
}

// nameDefaultsToDistributionUnlessChanged behaves like
// stringplanmodifier.UseStateForUnknown(), except it does not carry an
// omitted name's prior value forward when distribution is changing.
//
// In install mode, an omitted name defaults to distribution (see the name
// attribute's schema description). distribution is RequiresReplace, so
// changing it plans a brand-new resource -- but Terraform still proposes
// carrying computed attributes forward from the prior object unless a plan
// modifier says otherwise. Plain UseStateForUnknown() would therefore keep
// the OLD distribution's name across the replace; Create's own defaulting
// logic (`isKnownNonEmpty(plan.Name)`) then sees a non-empty planned name
// and skips filling in the new distribution's default, registering the new
// distribution under the previous one's name instead.
func nameDefaultsToDistributionUnlessChanged() planmodifier.String {
	return nameDefaultsToDistributionUnlessChangedModifier{}
}

type nameDefaultsToDistributionUnlessChangedModifier struct{}

func (m nameDefaultsToDistributionUnlessChangedModifier) Description(ctx context.Context) string {
	return m.MarkdownDescription(ctx)
}

func (m nameDefaultsToDistributionUnlessChangedModifier) MarkdownDescription(_ context.Context) string {
	return "Uses the prior state value for an unset name, except when distribution is changing, so a " +
		"replace does not carry the previous distribution's default name forward onto the new one."
}

func (m nameDefaultsToDistributionUnlessChangedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var stateDistribution, configDistribution types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("distribution"), &stateDistribution)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("distribution"), &configDistribution)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !configDistribution.IsUnknown() && configDistribution.ValueString() != stateDistribution.ValueString() {
		// distribution is changing: leave name Unknown so Create
		// recomputes its default against the NEW distribution instead of
		// inheriting the old one's.
		return
	}

	delegate := stringplanmodifier.UseStateForUnknown()
	delegate.PlanModifyString(ctx, req, resp)
}
