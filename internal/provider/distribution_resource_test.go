package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccWSLDistributionResource_InstallMode exercises the tar-less
// creation path (`wsl --install <Distribution>`, the same primitive
// everyday `wsl --install <Distribution>` uses) -- no rootfs file needed,
// and no existing registered distribution is touched. It needs:
//
//   - TF_ACC=1
//   - a real Windows host with WSL installed and Microsoft Store access
//     (install mode fetches the distribution from the Store)
//   - WSL_ACC_TEST_DISTRIBUTION set to a Store distribution identifier not
//     already registered on the host, e.g. "Ubuntu-24.04" (see
//     `wsl --list --online`); not set by default so this test, which can
//     be slow and needs network/Store access, only runs opt-in
//
// Run: TF_ACC=1 WSL_ACC_TEST_DISTRIBUTION=Ubuntu-24.04 go test ./internal/provider/... -v -run TestAccWSLDistributionResource_InstallMode
func TestAccWSLDistributionResource_InstallMode(t *testing.T) {
	distribution := os.Getenv("WSL_ACC_TEST_DISTRIBUTION")
	if distribution == "" {
		t.Skip("set WSL_ACC_TEST_DISTRIBUTION to a Store distribution identifier to run this acceptance test")
	}

	// EXPERIMENTAL / under verification: name is deliberately set to a
	// fixed value different from distribution here, to verify against a
	// real install whether wsl.exe's --name actually works (see
	// createInstall in internal/wsl/client.go). If it does not, revert
	// this test to name == distribution and restore the client.go/
	// ValidateConfig checks that used to require that.
	name := "testacc-wsl-distribution-install-test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDistributionInstallConfig(name, distribution),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("wsl_distribution.test", "name", name),
					resource.TestCheckResourceAttr("wsl_distribution.test", "distribution", distribution),
					resource.TestCheckResourceAttrSet("wsl_distribution.test", "version"),
					resource.TestCheckResourceAttrSet("wsl_distribution.test", "state"),
				),
			},
			{
				ResourceName:            "wsl_distribution.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"distribution"}, // unrecoverable on import; see docs/design-decisions/observable-state.md
			},
		},
	})
}

// TestAccWSLDistributionResource_ImportMode creates, updates (WSL version),
// and destroys a real WSL distribution via the import creation mode
// (rootfs + location). It needs:
//
//   - TF_ACC=1 (required by resource.Test itself)
//   - a real Windows host with WSL installed
//   - WSL_ACC_TEST_ROOTFS pointing at a small root filesystem tar to
//     import (a real distribution import can be large and slow, so no
//     rootfs is bundled with this repository; a minimal busybox-based
//     rootfs is a good, fast choice for this test specifically -- it does
//     not need to be bootable, since this test never launches the
//     distribution, only registers/reads/updates/unregisters it)
//
// Run: TF_ACC=1 WSL_ACC_TEST_ROOTFS=C:\path\to\rootfs.tar go test ./internal/provider/... -v -run TestAccWSLDistributionResource_ImportMode
func TestAccWSLDistributionResource_ImportMode(t *testing.T) {
	rootfs := os.Getenv("WSL_ACC_TEST_ROOTFS")
	if rootfs == "" {
		t.Skip("set WSL_ACC_TEST_ROOTFS to a root filesystem tar to run this acceptance test")
	}

	name := "testacc-wsl-distribution-import-test"
	location := filepath.Join(os.TempDir(), name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDistributionImportConfig(name, rootfs, location, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("wsl_distribution.test", "name", name),
					resource.TestCheckResourceAttr("wsl_distribution.test", "version", "2"),
					resource.TestCheckResourceAttrSet("wsl_distribution.test", "state"),
				),
			},
			{
				// WSL version change: this must be an in-place update, not
				// a replace -- see docs/design-decisions/update-vs-replace.md.
				Config: testAccDistributionImportConfig(name, rootfs, location, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("wsl_distribution.test", "version", "1"),
				),
			},
			{
				ResourceName:            "wsl_distribution.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rootfs", "location"}, // unrecoverable on import; see docs/design-decisions/observable-state.md
			},
		},
	})
}

func testAccDistributionInstallConfig(name, distribution string) string {
	return fmt.Sprintf(`
resource "wsl_distribution" "test" {
  name         = %[1]q
  distribution = %[2]q
}
`, name, distribution)
}

func testAccDistributionImportConfig(name, rootfs, location string, version int) string {
	return fmt.Sprintf(`
resource "wsl_distribution" "test" {
  name     = %[1]q
  rootfs   = %[2]q
  location = %[3]q
  version  = %[4]d
}
`, name, rootfs, location, version)
}
