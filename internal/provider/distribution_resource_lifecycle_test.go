package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jmnote/terraform-provider-wsl/internal/wsl"
)

type lifecycleRunner func([]string) (wsl.Result, error)

func (f lifecycleRunner) Run(_ context.Context, args ...string) (wsl.Result, error) {
	return f(args)
}

func lifecycleState(t *testing.T, model distributionResourceModel) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	(&distributionResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func lifecycleModel(t *testing.T, state tfsdk.State) distributionResourceModel {
	t.Helper()
	if !state.Raw.IsFullyKnown() || state.Raw.IsNull() {
		t.Fatalf("expected known, non-null state, got %s", state.Raw)
	}
	var model distributionResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func TestDistributionResource_UpdateResolvesCreationInputs(t *testing.T) {
	for _, mode := range []string{"install", "import", "adopt"} {
		t.Run(mode, func(t *testing.T) {
			prior := distributionResourceModel{
				Name: types.StringValue("worker"), Version: types.Int64Value(2), State: types.StringValue("Stopped"),
			}
			switch mode {
			case "install":
				prior.Distribution = types.StringValue("Ubuntu")
			case "import":
				prior.Rootfs = types.StringValue("rootfs.tar")
				prior.Location = types.StringValue(`C:\WSL\worker`)
			}
			planned := prior
			planned.Version = types.Int64Value(1)
			planned.State = types.StringUnknown()
			if mode == "import" {
				planned.Distribution = types.StringUnknown()
			} else {
				planned.Rootfs = types.StringUnknown()
				planned.Location = types.StringUnknown()
			}
			if mode == "adopt" {
				planned.Distribution = types.StringValue("Ubuntu")
			}
			state := lifecycleState(t, prior)
			plan := lifecycleState(t, planned)
			var calls [][]string
			r := &distributionResource{client: wsl.NewClient(lifecycleRunner(func(args []string) (wsl.Result, error) {
				calls = append(calls, args)
				return wsl.Result{Stdout: []byte("  NAME  STATE  VERSION\n  worker  Stopped  1\n")}, nil
			}))}
			resp := resource.UpdateResponse{State: state}
			r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Raw: plan.Raw, Schema: plan.Schema}}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			got := lifecycleModel(t, resp.State)
			want := prior
			want.Version = types.Int64Value(1)
			if mode == "adopt" {
				want.Distribution = types.StringValue("Ubuntu")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("state = %#v, want %#v", got, want)
			}
			if !reflect.DeepEqual(calls, [][]string{{"--set-version", "worker", "1"}, {"--list", "--verbose"}}) {
				t.Fatalf("unexpected operations: %v", calls)
			}
		})
	}
}

// TestDistributionResource_UpdatePreservesVersionOnRefreshFailure guards
// against a real bug: Update applied a version change to the real WSL
// distribution via SetVersion, but then discarded that change from state
// whenever the post-update refresh (used to also pick up the new "state"
// field) failed, because terraform-plugin-framework falls back to the
// prior state when Update returns without calling resp.State.Set. Update
// must persist the already-applied version even when the refresh fails.
func TestDistributionResource_UpdatePreservesVersionOnRefreshFailure(t *testing.T) {
	prior := distributionResourceModel{
		Name: types.StringValue("worker"), Distribution: types.StringValue("Ubuntu"),
		Version: types.Int64Value(1), State: types.StringValue("Stopped"),
	}
	planned := prior
	planned.Version = types.Int64Value(2)
	planned.State = types.StringUnknown()
	state := lifecycleState(t, prior)
	plan := lifecycleState(t, planned)

	setVersionCalled := false
	r := &distributionResource{client: wsl.NewClient(lifecycleRunner(func(args []string) (wsl.Result, error) {
		switch args[0] {
		case "--set-version":
			setVersionCalled = true
			return wsl.Result{}, nil
		case "--list":
			return wsl.Result{Stdout: []byte("Access is denied."), ExitCode: 1}, errors.New("read failed")
		default:
			t.Fatalf("unexpected command: %v", args)
		}
		return wsl.Result{}, nil
	}))}
	resp := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Raw: plan.Raw, Schema: plan.Schema}}, &resp)

	if !setVersionCalled || !resp.Diagnostics.HasError() {
		t.Fatalf("setVersionCalled = %v, diagnostics = %v", setVersionCalled, resp.Diagnostics)
	}
	got := lifecycleModel(t, resp.State)
	want := prior
	want.Version = types.Int64Value(2) // already applied via SetVersion; must not be lost
	want.State = types.StringNull()    // unobserved after the failed refresh
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("state = %#v, want %#v", got, want)
	}
}

func TestDistributionResource_CreatePreservesStateOnReadFailure(t *testing.T) {
	for _, mode := range []string{"install", "import"} {
		for _, version := range []types.Int64{types.Int64Unknown(), types.Int64Value(2)} {
			t.Run(mode+version.String(), func(t *testing.T) {
				planned := distributionResourceModel{
					Name: types.StringUnknown(), Distribution: types.StringValue("Ubuntu"),
					Rootfs: types.StringUnknown(), Location: types.StringUnknown(),
					Version: version, State: types.StringUnknown(),
				}
				if mode == "import" {
					planned.Name = types.StringValue("worker")
					planned.Distribution = types.StringUnknown()
					planned.Rootfs = types.StringValue("rootfs.tar")
					planned.Location = types.StringValue(`C:\WSL\worker`)
				}
				plan := lifecycleState(t, planned)
				created := false
				r := &distributionResource{client: wsl.NewClient(lifecycleRunner(func(args []string) (wsl.Result, error) {
					switch args[0] {
					case "--install", "--import":
						created = true
					case "--list":
						return wsl.Result{Stdout: []byte("Access is denied."), ExitCode: 1}, errors.New("read failed")
					case "--set-version":
					default:
						t.Fatalf("unexpected command: %v", args)
					}
					return wsl.Result{}, nil
				}))}
				resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
				r.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan{Raw: plan.Raw, Schema: plan.Schema}}, &resp)
				if !created || !resp.Diagnostics.HasError() {
					t.Fatalf("created = %v, diagnostics = %v", created, resp.Diagnostics)
				}
				got := lifecycleModel(t, resp.State)
				want := planned
				want.State = types.StringNull()
				if version.IsUnknown() {
					want.Version = types.Int64Null()
				}
				if mode == "install" {
					want.Name = types.StringValue("Ubuntu")
					want.Rootfs, want.Location = types.StringNull(), types.StringNull()
				} else {
					want.Distribution = types.StringNull()
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("state = %#v, want %#v", got, want)
				}
			})
		}
	}
}

func TestDistributionResource_ReadPreservesStateOnListFailure(t *testing.T) {
	state := lifecycleState(t, distributionResourceModel{
		Name: types.StringValue("worker"), Distribution: types.StringValue("Ubuntu"),
		Version: types.Int64Value(2), State: types.StringValue("Stopped"),
	})
	r := &distributionResource{client: wsl.NewClient(lifecycleRunner(func(args []string) (wsl.Result, error) {
		return wsl.Result{Stdout: []byte("WSL service unavailable."), ExitCode: 1}, errors.New("read failed")
	}))}
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if !resp.Diagnostics.HasError() || !resp.State.Raw.Equal(state.Raw) {
		t.Fatalf("expected read error and unchanged state, got diagnostics %v, state %s", resp.Diagnostics, resp.State.Raw)
	}
}
