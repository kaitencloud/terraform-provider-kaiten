// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const deploymentZoneResourceName = "kaiten_deployment_zone.test"

func TestAccDeploymentZone_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentZoneConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckDeploymentZoneExists(deploymentZoneResourceName),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("type"), knownvalue.StringExact("development")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("description"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("metadata"), knownvalue.Null()),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("slug"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:                         deploymentZoneResourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    testAccDeploymentZoneImportID,
				ImportStateVerifyIdentifierAttribute: "slug",
			},
		},
	})
}

func TestAccDeploymentZone_update(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	slug := "tf-acc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentZoneConfig_full(slug, rName, "staging", "first", `{ region = "eu-west-1" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckDeploymentZoneExists(deploymentZoneResourceName),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("slug"), knownvalue.StringExact(slug)),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("description"), knownvalue.StringExact("first")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("metadata"), knownvalue.StringExact(`{"region":"eu-west-1"}`)),
				},
			},
			{
				// Everything but the slug changes in place.
				Config: testAccDeploymentZoneConfig_full(slug, rName+"-renamed", "production", "second", `{ region = "us-east-1", tier = 2 }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(deploymentZoneResourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("name"), knownvalue.StringExact(rName+"-renamed")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("type"), knownvalue.StringExact("production")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("description"), knownvalue.StringExact("second")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("metadata"), knownvalue.StringExact(`{"region":"us-east-1","tier":2}`)),
				},
			},
			{
				// Dropping optional attributes clears them, still in place.
				Config: testAccDeploymentZoneConfig_slug(slug, rName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(deploymentZoneResourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("description"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("metadata"), knownvalue.Null()),
				},
			},
			{
				// A slug cannot be renamed, so a new one replaces the zone.
				Config: testAccDeploymentZoneConfig_slug(slug+"-new", rName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(deploymentZoneResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckDeploymentZoneExists(deploymentZoneResourceName),
					statecheck.ExpectKnownValue(deploymentZoneResourceName, tfjsonpath.New("slug"), knownvalue.StringExact(slug+"-new")),
				},
			},
		},
	})
}

func TestAccDeploymentZone_disappears(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentZoneConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckDeploymentZoneExists(deploymentZoneResourceName),
					stateCheckDeploymentZoneDisappears(deploymentZoneResourceName),
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestDeploymentZone_invalidSlug(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDeploymentZoneConfig_slug("Not_A_Slug", "invalid"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must be lowercase letters, digits and hyphens`),
			},
		},
	})
}

func testAccDeploymentZoneImportID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources[deploymentZoneResourceName]
	if !ok {
		return "", fmt.Errorf("not found in state: %s", deploymentZoneResourceName)
	}
	return rs.Primary.Attributes["slug"], nil
}

func testAccCheckDeploymentZoneDestroy(s *terraform.State) error {
	client, err := testAccClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kaiten_deployment_zone" {
			continue
		}
		slug := rs.Primary.Attributes["slug"]
		_, err := client.DeploymentZones.Get(context.Background(), slug)
		if err == nil {
			return fmt.Errorf("deployment zone %s still exists", slug)
		}
		if !isNotFound(err) {
			return err
		}
	}
	return nil
}

type deploymentZoneExistsCheck struct{ address string }

func (c deploymentZoneExistsCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	slug, err := stateSlug(req, c.address)
	if err != nil {
		resp.Error = err
		return
	}
	client, err := testAccClient()
	if err != nil {
		resp.Error = err
		return
	}
	if _, err := client.DeploymentZones.Get(ctx, slug); err != nil {
		resp.Error = fmt.Errorf("%s not found via the API: %w", c.address, err)
	}
}

func stateCheckDeploymentZoneExists(address string) statecheck.StateCheck {
	return deploymentZoneExistsCheck{address: address}
}

type deploymentZoneDisappearsCheck struct{ address string }

func (c deploymentZoneDisappearsCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	slug, err := stateSlug(req, c.address)
	if err != nil {
		resp.Error = err
		return
	}
	client, err := testAccClient()
	if err != nil {
		resp.Error = err
		return
	}
	resp.Error = client.DeploymentZones.Delete(ctx, slug)
}

func stateCheckDeploymentZoneDisappears(address string) statecheck.StateCheck {
	return deploymentZoneDisappearsCheck{address: address}
}

func stateSlug(req statecheck.CheckStateRequest, address string) (string, error) {
	r, err := stateResourceAtAddress(req.State, address)
	if err != nil {
		return "", err
	}
	slug, ok := r.AttributeValues["slug"].(string)
	if !ok || slug == "" {
		return "", fmt.Errorf("no slug in state for %s", address)
	}
	return slug, nil
}

func testAccDeploymentZoneConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "kaiten_deployment_zone" "test" {
  name = %[1]q
  type = "development"
}
`, rName)
}

func testAccDeploymentZoneConfig_slug(slug, rName string) string {
	return fmt.Sprintf(`
resource "kaiten_deployment_zone" "test" {
  slug = %[1]q
  name = %[2]q
  type = "development"
}
`, slug, rName)
}

func testAccDeploymentZoneConfig_full(slug, rName, zoneType, description, metadata string) string {
	return fmt.Sprintf(`
resource "kaiten_deployment_zone" "test" {
  slug        = %[1]q
  name        = %[2]q
  type        = %[3]q
  description = %[4]q
  metadata    = jsonencode(%[5]s)
}
`, slug, rName, zoneType, description, metadata)
}
