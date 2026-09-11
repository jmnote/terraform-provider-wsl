package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
