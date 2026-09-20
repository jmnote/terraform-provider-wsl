package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// nonNullObject is a minimal, non-null tftypes.Value standing in for a
// whole resource state/plan object. requiresReplaceUnlessImporting (like
// the stringplanmodifier.RequiresReplace it delegates to) only checks
// whether the *entire* state/plan object is null -- to distinguish
// resource creation/destroy from an update -- never its contents, so its
// shape does not need to match any real schema.
func nonNullObject() tftypes.Value {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"x": tftypes.String}}
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"x": tftypes.NewValue(tftypes.String, "placeholder"),
	})
}

func TestRequiresReplaceUnlessImporting(t *testing.T) {
	cases := []struct {
		name                string
		state, plan         types.String
		wantRequiresReplace bool
	}{
		{
			name:                "freshly imported (null state) adopts config without replace",
			state:               types.StringNull(),
			plan:                types.StringValue("C:\\images\\ubuntu.tar"),
			wantRequiresReplace: false,
		},
		{
			name:                "unchanged value never requires replace",
			state:               types.StringValue("C:\\images\\ubuntu.tar"),
			plan:                types.StringValue("C:\\images\\ubuntu.tar"),
			wantRequiresReplace: false,
		},
		{
			name:                "changed value requires replace",
			state:               types.StringValue("C:\\images\\ubuntu.tar"),
			plan:                types.StringValue("C:\\images\\debian.tar"),
			wantRequiresReplace: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State:      tfsdk.State{Raw: nonNullObject()},
				Plan:       tfsdk.Plan{Raw: nonNullObject()},
				StateValue: tc.state,
				PlanValue:  tc.plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}

			requiresReplaceUnlessImporting().PlanModifyString(context.Background(), req, resp)

			if resp.RequiresReplace != tc.wantRequiresReplace {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantRequiresReplace)
			}
		})
	}
}
