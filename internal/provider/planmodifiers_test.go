package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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

// TestNameDefaultsToDistributionUnlessChanged guards against a real bug:
// with plain stringplanmodifier.UseStateForUnknown, replacing a
// wsl_distribution by changing `distribution` (while `name` stays omitted,
// relying on its documented default) carried the OLD distribution's name
// forward into the plan for the new resource. Create then saw a non-empty
// planned name and skipped defaulting it to the NEW distribution, so the
// replacement got registered under the previous distribution's name.
func TestNameDefaultsToDistributionUnlessChanged(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&distributionResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	schema := schemaResp.Schema

	buildState := func(t *testing.T, model distributionResourceModel) tfsdk.State {
		t.Helper()
		state := tfsdk.State{Schema: schema}
		if diags := state.Set(context.Background(), &model); diags.HasError() {
			t.Fatal(diags)
		}
		return state
	}
	buildConfig := func(t *testing.T, model distributionResourceModel) tfsdk.Config {
		t.Helper()
		// tfsdk.Config has no Set method; build the Raw value via State.Set
		// (same underlying tftypes.Value shape) and wrap it as a Config.
		state := tfsdk.State{Schema: schema}
		if diags := state.Set(context.Background(), &model); diags.HasError() {
			t.Fatal(diags)
		}
		return tfsdk.Config{Raw: state.Raw, Schema: schema}
	}

	cases := []struct {
		name              string
		priorDistribution string
		newDistribution   string
		wantPlanValue     types.String
	}{
		{
			name:              "distribution unchanged: carries the defaulted name forward",
			priorDistribution: "Ubuntu-24.04",
			newDistribution:   "Ubuntu-24.04",
			wantPlanValue:     types.StringValue("Ubuntu-24.04"),
		},
		{
			name:              "distribution changing: does not carry the old default name forward",
			priorDistribution: "Ubuntu-24.04",
			newDistribution:   "Debian",
			wantPlanValue:     types.StringValue("Debian"),
		},
		{
			name:              "imported state with null distribution computes install default",
			priorDistribution: "Ubuntu-24.04",
			newDistribution:   "Ubuntu-24.04",
			wantPlanValue:     types.StringValue("Ubuntu-24.04"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stateDistribution := types.StringValue(tc.priorDistribution)
			if tc.name == "imported state with null distribution computes install default" {
				stateDistribution = types.StringNull()
			}
			state := buildState(t, distributionResourceModel{
				Name: types.StringValue(tc.priorDistribution), Distribution: stateDistribution,
			})
			// name omitted from config, same as relying on its documented default.
			config := buildConfig(t, distributionResourceModel{
				Name: types.StringNull(), Distribution: types.StringValue(tc.newDistribution),
			})

			req := planmodifier.StringRequest{
				State:       state,
				Config:      config,
				StateValue:  types.StringValue(tc.priorDistribution),
				ConfigValue: types.StringNull(),
				PlanValue:   types.StringUnknown(),
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

			nameDefaultsToDistributionUnlessChanged().PlanModifyString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.wantPlanValue) {
				t.Errorf("PlanValue = %#v, want %#v", resp.PlanValue, tc.wantPlanValue)
			}
		})
	}
}
