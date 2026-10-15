// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdk "github.com/kaitencloud/sdk-go"
)

// Attribute descriptions shared by the resource and the data sources, so the
// generated documentation says the same thing on every page.
const (
	descZoneID          = "Unique identifier (UUID) of the deployment zone, assigned by Kaiten."
	descZoneSlug        = "URL-friendly identifier of the deployment zone, unique within the organization."
	descZoneName        = "Human-readable name of the deployment zone, e.g. `AWS us-west-2`."
	descZoneType        = "Environment class of the deployment zone. Free-form, but `production`, `staging` and `development` are the values the Kaiten console labels, and feature-flag targeting rules match on it as `__kaiten.deploymentZone.type`."
	descZoneDescription = "Description of the deployment zone."
	descZoneMetadata    = "Metadata of the deployment zone, as a JSON-encoded object. Kaiten validates it against the metadata fields the organization declares for deployment zones."
	descZoneReleaseID   = "ID of the release currently deployed to the zone."
	descZoneCreatedAt   = "When the deployment zone was created, in RFC 3339 format."
	descZoneUpdatedAt   = "When the deployment zone was last updated, in RFC 3339 format."
)

// deploymentZoneModel is the Terraform shape of a deployment zone, shared by
// the resource and both data sources.
type deploymentZoneModel struct {
	ID          types.String         `tfsdk:"id"`
	Slug        types.String         `tfsdk:"slug"`
	Name        types.String         `tfsdk:"name"`
	Type        types.String         `tfsdk:"type"`
	Description types.String         `tfsdk:"description"`
	Metadata    jsontypes.Normalized `tfsdk:"metadata"`
	ReleaseID   types.String         `tfsdk:"release_id"`
	CreatedAt   timetypes.RFC3339    `tfsdk:"created_at"`
	UpdatedAt   timetypes.RFC3339    `tfsdk:"updated_at"`
}

// refresh copies zone into m. prior is the metadata Terraform already holds:
// when it means the same as what the API returned, it is kept verbatim, so
// key order and whitespace in the configuration never show up as drift.
func (m *deploymentZoneModel) refresh(zone sdk.DeploymentZone) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringPointerValue(zone.Id)
	m.Slug = types.StringPointerValue(zone.Slug)
	m.Name = types.StringValue(zone.Name)
	m.Type = types.StringValue(zone.Type)
	m.Description = types.StringValue(zone.Description)
	m.ReleaseID = types.StringPointerValue(zone.ReleaseId)
	m.CreatedAt = rfc3339(zone.CreatedAt)
	m.UpdatedAt = rfc3339(zone.UpdatedAt)

	metadata, err := flattenMetadata(zone.Metadata, m.Metadata)
	if err != nil {
		diags.AddError("Unable to read deployment zone metadata", err.Error())
		return diags
	}
	m.Metadata = metadata

	return diags
}

// input builds the SDK input for a create or an update from m.
func (m deploymentZoneModel) input() (sdk.DeploymentZoneInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	in := sdk.DeploymentZoneInput{
		Name:        m.Name.ValueString(),
		Type:        m.Type.ValueString(),
		Description: m.Description.ValueString(),
		Slug:        m.Slug.ValueStringPointer(),
		ReleaseID:   m.ReleaseID.ValueStringPointer(),
	}

	if !m.Metadata.IsNull() && !m.Metadata.IsUnknown() {
		diags.Append(m.Metadata.Unmarshal(&in.Metadata)...)
	}

	return in, diags
}

// flattenMetadata renders the API's metadata as JSON. An empty object reads
// as null unless Terraform already holds an equivalent value, so a zone with
// no metadata needs no metadata attribute in configuration.
func flattenMetadata(metadata *map[string]any, prior jsontypes.Normalized) (jsontypes.Normalized, error) {
	var value map[string]any
	if metadata != nil {
		value = *metadata
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return jsontypes.Normalized{}, err
	}
	if len(value) == 0 {
		encoded = []byte("{}")
	}
	current := jsontypes.NewNormalizedValue(string(encoded))

	if !prior.IsNull() && !prior.IsUnknown() {
		equal, diags := prior.StringSemanticEquals(context.Background(), current)
		if diags.HasError() {
			return jsontypes.Normalized{}, fmt.Errorf("comparing metadata: %v", diags)
		}
		if equal {
			return prior, nil
		}
	}
	if len(value) == 0 {
		return jsontypes.NewNormalizedNull(), nil
	}
	return current, nil
}

func rfc3339(t *time.Time) timetypes.RFC3339 {
	if t == nil {
		return timetypes.NewRFC3339Null()
	}
	return timetypes.NewRFC3339TimeValue(t.UTC())
}

// isNotFound reports whether err is the API saying the zone does not exist.
func isNotFound(err error) bool {
	var apiErr *sdk.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
