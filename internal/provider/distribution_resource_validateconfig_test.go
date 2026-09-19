package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// configValues builds a resource.ValidateConfigRequest for
// distributionResource, without needing a real Terraform run. Each of
// name/distribution/rootfs/location may be a string (a known value), nil
// (explicitly null/unset), or the sentinel unknownValue (references
// something not yet known, e.g. another resource's computed output).
type configValue any

var unknownValue = configValue(struct{}{})

func newValidateConfigRequest(t *testing.T, name, distribution, rootfs, location configValue) resource.ValidateConfigRequest {
	t.Helper()

	r := &distributionResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	objType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)

	toValue := func(attr string, v configValue) tftypes.Value {
		t := objType.AttributeTypes[attr]
		switch val := v.(type) {
		case string:
			return tftypes.NewValue(t, val)
		case nil:
			return tftypes.NewValue(t, nil)
		default:
			return tftypes.NewValue(t, tftypes.UnknownValue)
		}
	}

	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"name":         toValue("name", name),
		"distribution": toValue("distribution", distribution),
		"rootfs":       toValue("rootfs", rootfs),
		"location":     toValue("location", location),
		"version":      tftypes.NewValue(objType.AttributeTypes["version"], nil),
		"state":        tftypes.NewValue(objType.AttributeTypes["state"], nil),
	})

	return resource.ValidateConfigRequest{
		Config: tfsdk.Config{Raw: raw, Schema: schemaResp.Schema},
	}
}

func TestDistributionResource_ValidateConfig(t *testing.T) {
	cases := []struct {
		name                                    string
		cfgName, distribution, rootfs, location configValue
		wantError                               bool
	}{
		{
			name:         "install mode, name omitted: valid",
			distribution: "Ubuntu-24.04",
			wantError:    false,
		},
		{
			name:         "install mode, custom name: valid",
			cfgName:      "worker",
			distribution: "Ubuntu-24.04",
			wantError:    false,
		},
		{
			name:      "import mode, name set: valid",
			cfgName:   "worker",
			rootfs:    "C:\\r.tar",
			location:  "C:\\loc",
			wantError: false,
		},
		{
			name:         "conflicting mode: distribution + rootfs",
			distribution: "Ubuntu-24.04",
			rootfs:       "C:\\r.tar",
			location:     "C:\\loc",
			wantError:    true,
		},
		{
			name:      "missing mode: nothing set",
			wantError: true,
		},
		{
			name:      "incomplete import mode: rootfs without location",
			cfgName:   "worker",
			rootfs:    "C:\\r.tar",
			wantError: true,
		},
		{
			name:      "missing name in import mode",
			rootfs:    "C:\\r.tar",
			location:  "C:\\loc",
			wantError: true,
		},
		{
			// Guards the fix: an Unknown distribution (e.g. referencing
			// another resource's computed output) must not be treated as
			// "absent" and trigger a false "missing creation mode" error.
			name:         "unknown distribution defers validation",
			distribution: unknownValue,
			wantError:    false,
		},
		{
			// Same, for an Unknown name in otherwise-valid import mode.
			name:      "unknown name in import mode defers validation",
			cfgName:   unknownValue,
			rootfs:    "C:\\r.tar",
			location:  "C:\\loc",
			wantError: false,
		},
		{
			// An Unknown name must not hide a mode conflict that is already
			// completely known.
			name:         "unknown name still catches conflicting mode",
			cfgName:      unknownValue,
			distribution: "Ubuntu-24.04",
			rootfs:       "C:\\r.tar",
			location:     "C:\\loc",
			wantError:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &distributionResource{}
			req := newValidateConfigRequest(t, tc.cfgName, tc.distribution, tc.rootfs, tc.location)
			var resp resource.ValidateConfigResponse

			r.ValidateConfig(context.Background(), req, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Errorf("HasError() = %v, want %v; diagnostics: %v", got, tc.wantError, resp.Diagnostics)
			}
		})
	}
}
