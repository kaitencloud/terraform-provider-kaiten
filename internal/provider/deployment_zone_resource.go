// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdk "github.com/kaitencloud/sdk-go"
)

var (
	_ resource.Resource                = &deploymentZoneResource{}
	_ resource.ResourceWithConfigure   = &deploymentZoneResource{}
	_ resource.ResourceWithImportState = &deploymentZoneResource{}
)

// slugPattern is the API's own constraint on a slug, checked at plan time.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// NewDeploymentZoneResource returns the kaiten_deployment_zone resource.
func NewDeploymentZoneResource() resource.Resource {
	return &deploymentZoneResource{}
}

type deploymentZoneResource struct {
	client *sdk.Client
}

func (r *deploymentZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_zone"
}

func (r *deploymentZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Kaiten deployment zone: a region or environment that instances are deployed to.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: descZoneID,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: descZoneSlug + " Generated from `name` when omitted. " +
					"A slug cannot be renamed: changing it replaces the deployment zone.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 100),
					stringvalidator.RegexMatches(slugPattern, "must be lowercase letters, digits and hyphens, and start and end with a letter or digit"),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: descZoneName,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: descZoneType,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: descZoneDescription + " Defaults to an empty string.",
			},
			"metadata": schema.StringAttribute{
				CustomType:          jsontypes.NormalizedType{},
				Optional:            true,
				MarkdownDescription: descZoneMetadata + " Use `jsonencode()` to build it. Removing it clears the zone's metadata.",
			},
			"release_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: descZoneReleaseID + " Setting or changing it records a deployment of that release in the zone's history. " +
					"When omitted, Terraform leaves the release alone, so deployments made outside Terraform are not reverted.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				CustomType:          timetypes.RFC3339Type{},
				Computed:            true,
				MarkdownDescription: descZoneCreatedAt,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				CustomType:          timetypes.RFC3339Type{},
				Computed:            true,
				MarkdownDescription: descZoneUpdatedAt,
			},
		},
	}
}

func (r *deploymentZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*sdk.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *deploymentZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentZoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diags := plan.input()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.client.DeploymentZones.Create(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment zone",
			fmt.Sprintf("creating deployment zone (%s): %s", plan.Name.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(plan.refresh(zone)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.client.DeploymentZones.Get(ctx, state.Slug.ValueString())
	if isNotFound(err) {
		tflog.Warn(ctx, "deployment zone not found, removing from state", map[string]any{"slug": state.Slug.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment zone",
			fmt.Sprintf("reading deployment zone (%s): %s", state.Slug.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(state.refresh(zone)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update sends the whole zone: the API's PUT replaces it. The slug is part of
// the body only for the API to confirm it is unchanged; a new slug never
// reaches Update, since changing one forces replacement.
func (r *deploymentZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state deploymentZoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diags := plan.input()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := state.Slug.ValueString()
	if err := r.client.DeploymentZones.Update(ctx, slug, input); err != nil {
		resp.Diagnostics.AddError(
			"Error updating deployment zone",
			fmt.Sprintf("updating deployment zone (%s): %s", slug, err),
		)
		return
	}

	// The PUT answers 204 with no body, so read the zone back for the values
	// only the API knows, such as updated_at.
	zone, err := r.client.DeploymentZones.Get(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment zone",
			fmt.Sprintf("reading deployment zone (%s) after update: %s", slug, err),
		)
		return
	}

	resp.Diagnostics.Append(plan.refresh(zone)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deploymentZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeploymentZones.Delete(ctx, state.Slug.ValueString())
	if isNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting deployment zone",
			fmt.Sprintf("deleting deployment zone (%s): %s", state.Slug.ValueString(), err),
		)
	}
}

// ImportState imports a deployment zone by its slug, the identifier the API
// addresses zones by.
func (r *deploymentZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}
