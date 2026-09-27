// Package provider implements the skyhub Terraform provider.
package provider

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

var _ provider.Provider = &skyhubProvider{}

type skyhubProvider struct {
	version string
}

type providerModel struct {
	Endpoint        types.String `tfsdk:"endpoint"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	CredentialsFile types.String `tfsdk:"credentials_file"`
	Timeout         types.Int64  `tfsdk:"timeout"`
}

// New returns the provider constructor.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &skyhubProvider{version: version} }
}

func (p *skyhubProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "skyhub"
	resp.Version = p.version
}

func (p *skyhubProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a Sky Hub (Sagemcom) home router through its admin web UI. " +
			"The hub has no API: the provider drives the HTML forms with HTTP Digest auth and " +
			"serialises every request, so no `-parallelism` tuning is needed.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the hub. Defaults to `SKYHUB_URL`, then `http://192.168.50.1/`.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Admin user. Defaults to `SKYHUB_USER`, the credentials file, then `admin`.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Admin password. Defaults to `SKYHUB_PASSWORD`, then the credentials file.",
				Optional:            true,
				Sensitive:           true,
			},
			"credentials_file": schema.StringAttribute{
				MarkdownDescription: "File with `USER=` / `PASSWORD=` (and optional `URL=`) lines. Defaults to `~/skyhub`.",
				Optional:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Per-request timeout in seconds (default 30).",
				Optional:            true,
			},
		},
	}
}

func (p *skyhubProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint := firstNonEmpty(cfg.Endpoint.ValueString(), os.Getenv("SKYHUB_URL"))
	user := firstNonEmpty(cfg.Username.ValueString(), os.Getenv("SKYHUB_USER"))
	pass := firstNonEmpty(cfg.Password.ValueString(), os.Getenv("SKYHUB_PASSWORD"))
	if pass == "" || user == "" || endpoint == "" {
		path := cfg.CredentialsFile.ValueString()
		if path == "" {
			path = os.Getenv("SKYHUB_CREDENTIALS_FILE")
		}
		if path == "" {
			if home, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(home, "skyhub")
			}
		}
		if path != "" {
			fc, err := skyhub.ReadCredentialsFile(path)
			if err != nil && !os.IsNotExist(err) {
				resp.Diagnostics.AddError("Cannot read credentials file", err.Error())
				return
			}
			endpoint = firstNonEmpty(endpoint, fc.URL)
			user = firstNonEmpty(user, fc.User)
			pass = firstNonEmpty(pass, fc.Password)
		}
	}
	endpoint = firstNonEmpty(endpoint, skyhub.DefaultURL)
	user = firstNonEmpty(user, "admin")
	if pass == "" {
		resp.Diagnostics.AddError("Missing hub password",
			"Set the password attribute, SKYHUB_PASSWORD, or a PASSWORD= line in the credentials file.")
		return
	}
	timeout := 30 * time.Second
	if !cfg.Timeout.IsNull() && cfg.Timeout.ValueInt64() > 0 {
		timeout = time.Duration(cfg.Timeout.ValueInt64()) * time.Second
	}
	client, err := skyhub.New(endpoint, user, pass, skyhub.WithTimeout(timeout))
	if err != nil {
		resp.Diagnostics.AddError("Cannot create hub client", err.Error())
		return
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *skyhubProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newDHCPReservationResource,
		newServiceResource,
		newFirewallRuleResource,
		newLANResource,
		newWANResource,
		newUPnPResource,
		newALGResource,
		newEthernetResource,
		newFirewallGlobalsResource,
		newWirelessResource,
		newAdminPasswordResource,
	}
}

func (p *skyhubProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newAttachedDevicesDataSource,
		newSystemStatsDataSource,
		newWANStatusDataSource,
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// clientFrom extracts the configured client from provider data.
func clientFrom(data any) (*skyhub.Client, bool) {
	if data == nil {
		return nil, false
	}
	c, ok := data.(*skyhub.Client)
	return c, ok
}
