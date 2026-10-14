// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	sdk "github.com/kaitencloud/sdk-go"
)

var (
	_ datasource.DataSource              = &deploymentZonesDataSource{}
	_ datasource.DataSourceWithConfigure = &deploymentZonesDataSource{}
)

// NewDeploymentZonesDataSource returns the kaiten_deployment_zones data source.
func NewDeploymentZonesDataSource() datasource.DataSource {
	return &deploymentZonesDataSource{}
}

type deploymentZonesDataSource struct {
	client *sdk.Client
}

type deploymentZonesModel struct {
	DeploymentZones []deploymentZoneModel `tfsdk:"deployment_zones"`
}

func (d *deploymentZonesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_zones"
}

func (d *deploymentZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every deployment zone of the organization.",
		Attributes: map[string]schema.Attribute{
			"deployment_zones": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The organization's deployment zones, in the order the API returns them.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: deploymentZoneDataSourceAttributes(false),
				},
			},
		},
	}
}

func (d *deploymentZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *deploymentZonesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	zones, err := d.client.DeploymentZones.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing deployment zones", fmt.Sprintf("listing deployment zones: %s", err))
		return
	}

	data := deploymentZonesModel{DeploymentZones: make([]deploymentZoneModel, len(zones))}
	for i, zone := range zones {
		resp.Diagnostics.Append(data.DeploymentZones[i].refresh(zone)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
