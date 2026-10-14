// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	sdk "github.com/kaitencloud/sdk-go"
)

var (
	_ datasource.DataSource              = &deploymentZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &deploymentZoneDataSource{}
)

// NewDeploymentZoneDataSource returns the kaiten_deployment_zone data source.
func NewDeploymentZoneDataSource() datasource.DataSource {
	return &deploymentZoneDataSource{}
}

type deploymentZoneDataSource struct {
	client *sdk.Client
}

func (d *deploymentZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_zone"
}

func (d *deploymentZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads one Kaiten deployment zone, looked up by its slug.",
		Attributes:          deploymentZoneDataSourceAttributes(true),
	}
}

// deploymentZoneDataSourceAttributes is the read-only shape of a deployment
// zone. lookup makes slug the required lookup key of the singular data source;
// the plural one lists every attribute as computed.
func deploymentZoneDataSourceAttributes(lookup bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true, MarkdownDescription: descZoneID},
		"slug": schema.StringAttribute{
			Required:            lookup,
			Computed:            !lookup,
			MarkdownDescription: descZoneSlug,
		},
		"name":        schema.StringAttribute{Computed: true, MarkdownDescription: descZoneName},
		"type":        schema.StringAttribute{Computed: true, MarkdownDescription: descZoneType},
		"description": schema.StringAttribute{Computed: true, MarkdownDescription: descZoneDescription},
		"metadata": schema.StringAttribute{
			CustomType:          jsontypes.NormalizedType{},
			Computed:            true,
			MarkdownDescription: descZoneMetadata + " Null when the zone has none; decode it with `jsondecode()`.",
		},
		"release_id": schema.StringAttribute{Computed: true, MarkdownDescription: descZoneReleaseID},
		"created_at": schema.StringAttribute{CustomType: timetypes.RFC3339Type{}, Computed: true, MarkdownDescription: descZoneCreatedAt},
		"updated_at": schema.StringAttribute{CustomType: timetypes.RFC3339Type{}, Computed: true, MarkdownDescription: descZoneUpdatedAt},
	}
}

func (d *deploymentZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *deploymentZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data deploymentZoneModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := d.client.DeploymentZones.Get(ctx, data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment zone",
			fmt.Sprintf("reading deployment zone (%s): %s", data.Slug.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(data.refresh(zone)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// configureDataSourceClient extracts the client the provider configured. It
// returns nil before the provider is configured, as during validation.
func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *sdk.Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*sdk.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sdk.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return client
}
