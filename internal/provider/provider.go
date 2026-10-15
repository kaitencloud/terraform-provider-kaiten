// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

// Package provider implements the Kaiten Terraform provider.
package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdk "github.com/kaitencloud/sdk-go"
)

var _ provider.Provider = &kaitenProvider{}

// New returns the provider factory consumed by main.go.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &kaitenProvider{version: version, resolver: newResolver()}
	}
}

type kaitenProvider struct {
	version  string
	resolver resolver
}

type kaitenProviderModel struct {
	BaseURL   types.String `tfsdk:"base_url"`
	AuthToken types.String `tfsdk:"auth_token"`
}

func (p *kaitenProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kaiten"
	resp.Version = p.version
}

func (p *kaitenProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage [Kaiten](https://kaiten.sh) infrastructure through the Kaiten Core API.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Base URL of the Kaiten Core API, including its path prefix, " +
					"e.g. `https://kaiten.example.com/api`. May also be set via the `" + envBaseURL +
					"` environment variable, or the `base_url` key of the Kaiten CLI configuration file.",
			},
			"auth_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "Organization-scoped API token (`ksh_...`). May also be set via the `" + envAuthToken +
					"` environment variable, or the `auth_token` key of the Kaiten CLI configuration file.",
			},
		},
	}
}

func (p *kaitenProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config kaitenProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A value wired to another resource's output is unknown during planning;
	// treating it as empty would silently fall through to another source.
	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Kaiten base URL",
			"base_url depends on a value known only after apply. Set a static value, or use the "+envBaseURL+" environment variable.",
		)
	}
	if config.AuthToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_token"),
			"Unknown Kaiten auth token",
			"auth_token depends on a value known only after apply. Set a static value, or use the "+envAuthToken+" environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resolved, warnings, err := p.resolver.resolve(config.BaseURL.ValueString(), config.AuthToken.ValueString())
	for _, w := range warnings {
		resp.Diagnostics.AddWarning("Insecure Kaiten CLI configuration file", w)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure the Kaiten provider", err.Error())
		return
	}
	tflog.Debug(ctx, "resolved Kaiten connection settings", map[string]any{
		"base_url":          resolved.BaseURL,
		"base_url_source":   resolved.BaseURLSource,
		"auth_token_source": resolved.AuthTokenSource,
	})

	userAgent := "terraform-provider-kaiten/" + p.version
	client, err := sdk.NewClient(resolved.BaseURL,
		sdk.WithBearerToken(resolved.AuthToken),
		sdk.WithRequestEditorFn(func(_ context.Context, r *http.Request) error {
			r.Header.Set("User-Agent", userAgent)
			return nil
		}),
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create the Kaiten API client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *kaitenProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDeploymentZoneResource,
	}
}

func (p *kaitenProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDeploymentZoneDataSource,
		NewDeploymentZonesDataSource,
	}
}
