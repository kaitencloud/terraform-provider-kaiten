// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDeploymentZoneDataSource_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	dataSourceName := "data.kaiten_deployment_zone.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentZoneDataSourceConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("metadata"), knownvalue.StringExact(`{"region":"eu-west-1"}`)),
					statecheck.CompareValuePairs(dataSourceName, tfjsonpath.New("id"), deploymentZoneResourceName, tfjsonpath.New("id"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceName, tfjsonpath.New("type"), deploymentZoneResourceName, tfjsonpath.New("type"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceName, tfjsonpath.New("description"), deploymentZoneResourceName, tfjsonpath.New("description"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceName, tfjsonpath.New("created_at"), deploymentZoneResourceName, tfjsonpath.New("created_at"), compare.ValuesSame()),
				},
			},
		},
	})
}

func TestAccDeploymentZoneDataSource_notFound(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "kaiten_deployment_zone" "test" {
  slug = "tf-acc-does-not-exist"
}
`,
				ExpectError: regexp.MustCompile(`Error reading deployment zone`),
			},
		},
	})
}

func TestAccDeploymentZonesDataSource_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	// Not parallel: the list covers the whole organization, so zones the
	// other tests create and destroy would change it between apply and the
	// plan that follows, and the test would report drift that is not there.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentZonesDataSourceConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					// Other zones may exist in the organization, so look for
					// this test's own zone rather than counting.
					statecheck.ExpectKnownOutputValue("names", knownvalue.SetPartial([]knownvalue.Check{
						knownvalue.StringExact(rName),
					})),
				},
			},
		},
	})
}

func testAccDeploymentZoneDataSourceConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "kaiten_deployment_zone" "test" {
  name     = %[1]q
  type     = "development"
  metadata = jsonencode({ region = "eu-west-1" })
}

data "kaiten_deployment_zone" "test" {
  slug = kaiten_deployment_zone.test.slug
}
`, rName)
}

func testAccDeploymentZonesDataSourceConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "kaiten_deployment_zone" "test" {
  name = %[1]q
  type = "development"
}

data "kaiten_deployment_zones" "test" {
  depends_on = [kaiten_deployment_zone.test]
}

output "names" {
  value = toset([for zone in data.kaiten_deployment_zones.test.deployment_zones : zone.name])
}
`, rName)
}
