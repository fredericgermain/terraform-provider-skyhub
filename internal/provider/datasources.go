package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

type dsConfigured struct{ client *skyhub.Client }

func (d *dsConfigured) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := clientFrom(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "expected *skyhub.Client")
		return
	}
	d.client = c
}

// ---------- skyhub_attached_devices ----------

type attachedDevicesDataSource struct{ dsConfigured }

type attachedDeviceModel struct {
	MAC      types.String `tfsdk:"mac"`
	Hostname types.String `tfsdk:"hostname"`
	IPv4     types.String `tfsdk:"ipv4"`
	DHCPName types.String `tfsdk:"dhcp_name"`
	IPv6     types.String `tfsdk:"ipv6"`
}

type attachedDevicesModel struct {
	ID      types.String          `tfsdk:"id"`
	Devices []attachedDeviceModel `tfsdk:"devices"`
}

func newAttachedDevicesDataSource() datasource.DataSource { return &attachedDevicesDataSource{} }

func (d *attachedDevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_attached_devices"
}

func (d *attachedDevicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Devices currently known to the hub (Attached Devices page).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, MarkdownDescription: "Always `attached_devices`."},
			"devices": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "One entry per device.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"mac":       schema.StringAttribute{Computed: true, MarkdownDescription: "MAC address."},
					"hostname":  schema.StringAttribute{Computed: true, MarkdownDescription: "Hostname, or `UNKNOWN`."},
					"ipv4":      schema.StringAttribute{Computed: true, MarkdownDescription: "IPv4 address."},
					"dhcp_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Name from the DHCP lease, if any."},
					"ipv6":      schema.StringAttribute{Computed: true, MarkdownDescription: "IPv6 address, if any."},
				}},
			},
		},
	}
}

func (d *attachedDevicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	devs, err := d.client.AttachedDevices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read attached devices", err.Error())
		return
	}
	m := attachedDevicesModel{ID: types.StringValue("attached_devices"), Devices: []attachedDeviceModel{}}
	for _, x := range devs {
		m.Devices = append(m.Devices, attachedDeviceModel{
			MAC: types.StringValue(x.MAC.String()), Hostname: types.StringValue(x.Hostname), IPv4: types.StringValue(x.IPv4.String()),
			DHCPName: types.StringValue(x.DHCPName), IPv6: types.StringValue(x.IPv6.String()),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// ---------- skyhub_system_stats ----------

type systemStatsDataSource struct{ dsConfigured }

type portStatsModel struct {
	Key        types.String `tfsdk:"key"`
	Name       types.String `tfsdk:"name"`
	Status     types.String `tfsdk:"status"`
	TxPkts     types.Int64  `tfsdk:"tx_pkts"`
	RxPkts     types.Int64  `tfsdk:"rx_pkts"`
	Collisions types.Int64  `tfsdk:"collisions"`
	TxBps      types.Int64  `tfsdk:"tx_bps"`
	RxBps      types.Int64  `tfsdk:"rx_bps"`
	UptimeS    types.Int64  `tfsdk:"uptime_seconds"`
}

type systemStatsModel struct {
	ID                types.String     `tfsdk:"id"`
	UptimeSeconds     types.Int64      `tfsdk:"uptime_seconds"`
	Ports             []portStatsModel `tfsdk:"ports"`
	DSLDownKbps       types.Int64      `tfsdk:"dsl_down_kbps"`
	DSLUpKbps         types.Int64      `tfsdk:"dsl_up_kbps"`
	NoiseMarginDownDB types.Float64    `tfsdk:"noise_margin_down_db"`
	NoiseMarginUpDB   types.Float64    `tfsdk:"noise_margin_up_db"`
	AttenuationDownDB types.Float64    `tfsdk:"attenuation_down_db"`
}

func newSystemStatsDataSource() datasource.DataSource { return &systemStatsDataSource{} }

func (d *systemStatsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_stats"
}

func (d *systemStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Uptime, per-port counters and DSL line statistics (System page).",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true, MarkdownDescription: "Always `system_stats`."},
			"uptime_seconds": schema.Int64Attribute{Computed: true, MarkdownDescription: "System uptime in seconds."},
			"ports": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "wan, lan, wlan24 and wlan5 rows.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"key":            schema.StringAttribute{Computed: true, MarkdownDescription: "`wan`, `lan`, `wlan24` or `wlan5`."},
					"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Row label."},
					"status":         schema.StringAttribute{Computed: true, MarkdownDescription: "Link status."},
					"tx_pkts":        schema.Int64Attribute{Computed: true},
					"rx_pkts":        schema.Int64Attribute{Computed: true},
					"collisions":     schema.Int64Attribute{Computed: true},
					"tx_bps":         schema.Int64Attribute{Computed: true},
					"rx_bps":         schema.Int64Attribute{Computed: true},
					"uptime_seconds": schema.Int64Attribute{Computed: true},
				}},
			},
			"dsl_down_kbps":        schema.Int64Attribute{Computed: true, MarkdownDescription: "Downstream sync rate (0 when not synced)."},
			"dsl_up_kbps":          schema.Int64Attribute{Computed: true, MarkdownDescription: "Upstream sync rate."},
			"noise_margin_down_db": schema.Float64Attribute{Computed: true},
			"noise_margin_up_db":   schema.Float64Attribute{Computed: true},
			"attenuation_down_db":  schema.Float64Attribute{Computed: true, MarkdownDescription: "First downstream band attenuation."},
		},
	}
}

func (d *systemStatsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	s, err := d.client.SystemStats(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read system stats", err.Error())
		return
	}
	m := systemStatsModel{ID: types.StringValue("system_stats"), UptimeSeconds: types.Int64Value(int64(s.Uptime.Std().Seconds())), Ports: []portStatsModel{}}
	for _, p := range s.Ports {
		m.Ports = append(m.Ports, portStatsModel{
			Key: types.StringValue(p.Key), Name: types.StringValue(p.Name), Status: types.StringValue(p.Status),
			TxPkts: types.Int64Value(int64(p.TxPkts)), RxPkts: types.Int64Value(int64(p.RxPkts)), Collisions: types.Int64Value(int64(p.Collisions)),
			TxBps: types.Int64Value(int64(p.TxBps)), RxBps: types.Int64Value(int64(p.RxBps)), UptimeS: types.Int64Value(int64(p.Uptime.Std().Seconds())),
		})
	}
	m.DSLDownKbps, m.DSLUpKbps = types.Int64Value(0), types.Int64Value(0)
	m.NoiseMarginDownDB, m.NoiseMarginUpDB, m.AttenuationDownDB = types.Float64Value(0), types.Float64Value(0), types.Float64Value(0)
	if s.DSL != nil {
		m.DSLDownKbps = types.Int64Value(int64(s.DSL.DownKbps))
		m.DSLUpKbps = types.Int64Value(int64(s.DSL.UpKbps))
		m.NoiseMarginDownDB = types.Float64Value(s.DSL.NoiseMarginDownDB)
		m.NoiseMarginUpDB = types.Float64Value(s.DSL.NoiseMarginUpDB)
		if len(s.DSL.AttenuationDownDB) > 0 {
			m.AttenuationDownDB = types.Float64Value(s.DSL.AttenuationDownDB[0])
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// ---------- skyhub_wan_status ----------

type wanStatusDataSource struct{ dsConfigured }

type wanStatusModel struct {
	ID              types.String `tfsdk:"id"`
	Up              types.Bool   `tfsdk:"up"`
	Protocol        types.String `tfsdk:"protocol"`
	Interface       types.String `tfsdk:"interface"`
	IPv4            types.String `tfsdk:"ipv4"`
	Netmask         types.String `tfsdk:"netmask"`
	Gateway         types.String `tfsdk:"gateway"`
	MAC             types.String `tfsdk:"mac"`
	DNS             types.List   `tfsdk:"dns"`
	UptimeSeconds   types.Int64  `tfsdk:"uptime_seconds"`
	IPv6            types.String `tfsdk:"ipv6"`
	IPv6Gateway     types.String `tfsdk:"ipv6_gateway"`
	DelegatedPrefix types.String `tfsdk:"delegated_prefix"`
	ModemState      types.String `tfsdk:"modem_state"`
	RouterMode      types.String `tfsdk:"router_mode"`
	Firmware        types.String `tfsdk:"firmware"`
	DSLFirmware     types.String `tfsdk:"dsl_firmware"`
}

func newWANStatusDataSource() datasource.DataSource { return &wanStatusDataSource{} }

func (d *wanStatusDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wan_status"
}

func (d *wanStatusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	str := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "WAN link state, addressing and firmware version.",
		Attributes: map[string]schema.Attribute{
			"id":               str("Always `wan_status`."),
			"up":               schema.BoolAttribute{Computed: true, MarkdownDescription: "WAN connection is up."},
			"protocol":         str("e.g. `ipoe`."),
			"interface":        str("WAN interface, e.g. `ptm0.1`."),
			"ipv4":             str("Public IPv4 address."),
			"netmask":          str("WAN netmask."),
			"gateway":          str("IPv4 gateway."),
			"mac":              str("WAN MAC address."),
			"dns":              schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Upstream DNS servers."},
			"uptime_seconds":   schema.Int64Attribute{Computed: true, MarkdownDescription: "WAN uptime in seconds."},
			"ipv6":             str("WAN IPv6 address with prefix length."),
			"ipv6_gateway":     str("IPv6 gateway (link-local)."),
			"delegated_prefix": str("Delegated IPv6 prefix."),
			"modem_state":      str("DSL modem state."),
			"router_mode":      str("`ADSL`, `WANOE` or `AUTO`."),
			"firmware":         str("Hub firmware version."),
			"dsl_firmware":     str("DSL modem firmware version."),
		},
	}
}

func (d *wanStatusDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	w, err := d.client.WANStatus(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read WAN status", err.Error())
		return
	}
	info, err := d.client.RouterInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read router info", err.Error())
		return
	}
	dns := make([]string, 0, len(w.DNS))
	for _, a := range w.DNS {
		dns = append(dns, a.String())
	}
	dnsList, diags := types.ListValueFrom(ctx, types.StringType, dns)
	resp.Diagnostics.Append(diags...)
	m := wanStatusModel{
		ID: types.StringValue("wan_status"), Up: types.BoolValue(w.Up), Protocol: types.StringValue(w.Protocol),
		Interface: types.StringValue(w.Interface), IPv4: types.StringValue(w.IPv4.String()), Netmask: types.StringValue(w.Netmask.String()),
		Gateway: types.StringValue(w.Gateway.String()), MAC: types.StringValue(w.MAC.String()), DNS: dnsList,
		UptimeSeconds: types.Int64Value(int64(w.Uptime.Std().Seconds())), IPv6: types.StringValue(w.IPv6.String()),
		IPv6Gateway: types.StringValue(w.IPv6Gateway.String()), DelegatedPrefix: types.StringValue(w.DelegatedPrefix.String()),
		ModemState: types.StringValue(w.ModemState), RouterMode: types.StringValue(w.RouterMode),
		Firmware: types.StringValue(info.Firmware), DSLFirmware: types.StringValue(info.DSLFirmware),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
