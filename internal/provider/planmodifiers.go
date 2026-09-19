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

// nameDefaultsToDistributionUnlessChanged computes an omitted install-mode
// name from distribution and otherwise behaves like
// stringplanmodifier.UseStateForUnknown().
//
// Computing this value in the modifier also avoids treating a freshly
// imported state's null distribution as a change, which would otherwise
// leave name Unknown and make the following RequiresReplace modifier plan a
// needless replacement. Create retains a backstop for a distribution that is
// still unknown at planning time.
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
	// An explicitly configured name, including an unknown expression, wins.
	// Do not overwrite an expression that Terraform must resolve later.
	if !req.ConfigValue.IsNull() {
		return
	}

	var configDistribution types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("distribution"), &configDistribution)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if configDistribution.IsUnknown() {
		// The default cannot be computed until the distribution expression is
		// known. Create retains the same defaulting logic as an apply-time
		// backstop for this case.
		return
	}
	if isKnownNonEmpty(configDistribution) {
		// Compute the documented default directly. This is important after
		// import: state.distribution is null, so comparing state and config
		// distributions would otherwise leave name unknown and the following
		// RequiresReplace modifier would incorrectly force replacement.
		resp.PlanValue = types.StringValue(configDistribution.ValueString())
		return
	}

	delegate := stringplanmodifier.UseStateForUnknown()
	delegate.PlanModifyString(ctx, req, resp)
}
