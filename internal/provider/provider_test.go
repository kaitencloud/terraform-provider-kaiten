// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-json"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdk "github.com/kaitencloud/sdk-go"
)

// testAccProtoV6ProviderFactories serves the provider in-process to the
// Terraform CLI that acceptance tests drive.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kaiten": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck fails fast when the provider could not connect: acceptance
// tests resolve their settings exactly as the provider does.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if _, _, err := newResolver().resolve("", ""); err != nil {
		t.Fatalf("acceptance tests need a Kaiten API: %v", err)
	}
}

// testAccClient builds an API client the way the provider does, for checks
// that look at the API directly.
func testAccClient() (*sdk.Client, error) {
	s, _, err := newResolver().resolve("", "")
	if err != nil {
		return nil, err
	}
	return sdk.NewClient(s.BaseURL, sdk.WithBearerToken(s.AuthToken))
}

func TestProviderSchemaIsValid(t *testing.T) {
	var resp fwprovider.SchemaResponse
	New("test")().Schema(context.Background(), fwprovider.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("provider schema implementation: %v", diags)
	}
}

// stateResourceAtAddress finds a resource in the state Terraform reports.
func stateResourceAtAddress(state *tfjson.State, address string) (*tfjson.StateResource, error) {
	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		return nil, fmt.Errorf("no state available")
	}
	for _, r := range state.Values.RootModule.Resources {
		if r.Address == address {
			return r, nil
		}
	}
	return nil, fmt.Errorf("not found in state: %s", address)
}
