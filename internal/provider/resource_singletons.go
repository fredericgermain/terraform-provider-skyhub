package provider

import (
	"context"
	"errors"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

func optString(desc string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{MarkdownDescription: desc, Optional: true, Computed: true, Validators: validators,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
}

func optBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{MarkdownDescription: desc, Optional: true, Computed: true,
		PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
}

func optInt(desc string, validators ...validator.Int64) schema.Int64Attribute {
	return schema.Int64Attribute{MarkdownDescription: desc, Optional: true, Computed: true, Validators: validators,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}}
}

func addr(s types.String) skyhub.Addr {
	a, _ := netip.ParseAddr(s.ValueString())
	return skyhub.Addr{Addr: a}
}

// ================= skyhub_upnp =================

var (
	_ resource.ResourceWithConfigure   = &upnpResource{}
	_ resource.ResourceWithImportState = &upnpResource{}
)

type upnpResource struct{ configured }

type upnpModel struct {
	ID                types.String `tfsdk:"id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	AdvertiseInterval types.Int64  `tfsdk:"advertise_interval"`
	AdvertiseTTL      types.Int64  `tfsdk:"advertise_ttl"`
}

func newUPnPResource() resource.Resource { return &upnpResource{} }

func (r *upnpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_upnp"
}

func (r *upnpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "UPnP settings (singleton, id `default`). Import with `terraform import skyhub_upnp.this default`.",
		Attributes: map[string]schema.Attribute{
			"id":                 singletonIDAttribute(),
			"enabled":            schema.BoolAttribute{MarkdownDescription: "Turn UPnP on or off.", Required: true},
			"advertise_interval": optInt("Advertisement period in minutes (1-1440, hub default 30).", int64validator.Between(1, 1440)),
			"advertise_ttl":      optInt("Advertisement TTL in hops (1-255, hub default 4).", int64validator.Between(1, 255)),
		},
	}
}

func (r *upnpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *upnpResource) read(ctx context.Context, m *upnpModel) error {
	u, err := r.client.UPnPConfig(ctx)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(singletonID)
	m.Enabled = types.BoolValue(u.Enabled)
	m.AdvertiseInterval = types.Int64Value(int64(u.AdvertiseInterval))
	m.AdvertiseTTL = types.Int64Value(int64(u.AdvertiseTTL))
	return nil
}

func (r *upnpResource) apply(ctx context.Context, m *upnpModel) error {
	cur, err := r.client.UPnPConfig(ctx)
	if err != nil {
		return err
	}
	want := skyhub.UPnPConfig{
		Enabled:           m.Enabled.ValueBool(),
		AdvertiseInterval: int(int64Or(m.AdvertiseInterval, int64(cur.AdvertiseInterval))),
		AdvertiseTTL:      int(int64Or(m.AdvertiseTTL, int64(cur.AdvertiseTTL))),
	}
	if want.Enabled != cur.Enabled || want.AdvertiseInterval != cur.AdvertiseInterval || want.AdvertiseTTL != cur.AdvertiseTTL {
		if err := r.client.SetUPnP(ctx, want); err != nil {
			return err
		}
	}
	return r.read(ctx, m)
}

func (r *upnpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m upnpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set UPnP", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *upnpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m upnpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read UPnP", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *upnpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m upnpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set UPnP", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *upnpResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_upnp")
}

func (r *upnpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	singletonImport(ctx, req, resp)
}

// ================= skyhub_alg =================

var (
	_ resource.ResourceWithConfigure   = &algResource{}
	_ resource.ResourceWithImportState = &algResource{}
)

type algResource struct{ configured }

type algModel struct {
	ID   types.String `tfsdk:"id"`
	SIP  types.Bool   `tfsdk:"sip"`
	H323 types.Bool   `tfsdk:"h323"`
}

func newALGResource() resource.Resource { return &algResource{} }

func (r *algResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alg"
}

func (r *algResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "SIP and H.323 application layer gateways (singleton, id `default`).",
		Attributes: map[string]schema.Attribute{
			"id":   singletonIDAttribute(),
			"sip":  schema.BoolAttribute{MarkdownDescription: "SIP ALG.", Required: true},
			"h323": schema.BoolAttribute{MarkdownDescription: "H.323 ALG.", Required: true},
		},
	}
}

func (r *algResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *algResource) read(ctx context.Context, m *algModel) error {
	a, err := r.client.ALGConfig(ctx)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(singletonID)
	m.SIP = types.BoolValue(a.SIP)
	m.H323 = types.BoolValue(a.H323)
	return nil
}

func (r *algResource) apply(ctx context.Context, m *algModel) error {
	cur, err := r.client.ALGConfig(ctx)
	if err != nil {
		return err
	}
	want := skyhub.ALGConfig{SIP: m.SIP.ValueBool(), H323: m.H323.ValueBool()}
	if want != *cur {
		if err := r.client.SetALG(ctx, want); err != nil {
			return err
		}
	}
	return r.read(ctx, m)
}

func (r *algResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m algModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set ALG", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *algResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m algModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read ALG", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *algResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m algModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set ALG", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *algResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_alg")
}

func (r *algResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	singletonImport(ctx, req, resp)
}

// ================= skyhub_ethernet =================

var (
	_ resource.ResourceWithConfigure   = &ethernetResource{}
	_ resource.ResourceWithImportState = &ethernetResource{}
)

type ethernetResource struct{ configured }

type ethernetModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	EEE  types.Bool   `tfsdk:"eee"`
}

func newEthernetResource() resource.Resource { return &ethernetResource{} }

func (r *ethernetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ethernet"
}

func (r *ethernetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "LAN Ethernet port mode (singleton, id `default`). **Changing a value reboots the hub.** " +
			"When the configured values already match, nothing is sent.",
		Attributes: map[string]schema.Attribute{
			"id":   singletonIDAttribute(),
			"type": optString("`Gigabit` or `Fast`.", stringvalidator.OneOf("Gigabit", "Fast")),
			"eee":  optBool("Energy Efficient Ethernet."),
		},
	}
}

func (r *ethernetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *ethernetResource) read(ctx context.Context, m *ethernetModel) error {
	e, err := r.client.EthernetConfig(ctx)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(singletonID)
	m.Type = types.StringValue(e.Type)
	m.EEE = types.BoolValue(e.EEE)
	return nil
}

func (r *ethernetResource) apply(ctx context.Context, m *ethernetModel) error {
	cur, err := r.client.EthernetConfig(ctx)
	if err != nil {
		return err
	}
	want := skyhub.EthernetConfig{Type: strOr(m.Type, cur.Type), EEE: boolOr(m.EEE, cur.EEE)}
	if want != *cur {
		if err := r.client.SetEthernet(ctx, want); err != nil {
			return err
		}
	}
	return r.read(ctx, m)
}

func (r *ethernetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m ethernetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set Ethernet", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *ethernetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m ethernetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read Ethernet", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *ethernetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m ethernetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set Ethernet", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *ethernetResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_ethernet")
}

func (r *ethernetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	singletonImport(ctx, req, resp)
}

// ================= skyhub_wan =================

var (
	_ resource.ResourceWithConfigure   = &wanResource{}
	_ resource.ResourceWithImportState = &wanResource{}
)

type wanResource struct{ configured }

type wanModel struct {
	ID             types.String `tfsdk:"id"`
	RouterMode     types.String `tfsdk:"router_mode"`
	MTU            types.Int64  `tfsdk:"mtu"`
	DMZEnabled     types.Bool   `tfsdk:"dmz_enabled"`
	DMZIP          types.String `tfsdk:"dmz_ip"`
	RespondToPing  types.Bool   `tfsdk:"respond_to_ping"`
	RespondToPing6 types.Bool   `tfsdk:"respond_to_ping6"`
}

func newWANResource() resource.Resource { return &wanResource{} }

func (r *wanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wan"
}

func (r *wanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "WAN setup: router mode, MTU, DMZ host and WAN ping responses (singleton, id `default`). " +
			"Unset attributes keep the hub's current value. Saving re-applies the WAN configuration.",
		Attributes: map[string]schema.Attribute{
			"id":               singletonIDAttribute(),
			"router_mode":      optString("`ADSL`, `WANOE` or `AUTO`.", stringvalidator.OneOf("ADSL", "WANOE", "AUTO")),
			"mtu":              optInt("MTU (1280-1534).", int64validator.Between(1280, 1534)),
			"dmz_enabled":      optBool("Expose a LAN host as DMZ."),
			"dmz_ip":           optString("DMZ host IPv4 address (required when `dmz_enabled`)."),
			"respond_to_ping":  optBool("Answer ICMP echo on the WAN IPv4 address."),
			"respond_to_ping6": optBool("Answer ICMPv6 echo on the WAN IPv6 address."),
		},
	}
}

func (r *wanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *wanResource) read(ctx context.Context, m *wanModel) error {
	w, err := r.client.WANConfig(ctx)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(singletonID)
	m.RouterMode = types.StringValue(w.RouterMode)
	m.MTU = types.Int64Value(int64(w.MTU))
	m.DMZEnabled = types.BoolValue(w.DMZEnabled)
	m.DMZIP = types.StringValue(w.DMZIP.String())
	m.RespondToPing = types.BoolValue(w.RespondToPing)
	m.RespondToPing6 = types.BoolValue(w.RespondToPing6)
	return nil
}

func (r *wanResource) apply(ctx context.Context, m *wanModel) error {
	cur, err := r.client.WANConfig(ctx)
	if err != nil {
		return err
	}
	want := skyhub.WANConfig{
		RouterMode:     strOr(m.RouterMode, cur.RouterMode),
		MTU:            int(int64Or(m.MTU, int64(cur.MTU))),
		DMZEnabled:     boolOr(m.DMZEnabled, cur.DMZEnabled),
		DMZIP:          cur.DMZIP,
		DMZIPv6:        cur.DMZIPv6,
		RespondToPing:  boolOr(m.RespondToPing, cur.RespondToPing),
		RespondToPing6: boolOr(m.RespondToPing6, cur.RespondToPing6),
	}
	if !m.DMZIP.IsNull() && !m.DMZIP.IsUnknown() && m.DMZIP.ValueString() != "" {
		want.DMZIP = addr(m.DMZIP)
	}
	if want.RouterMode != cur.RouterMode || want.MTU != cur.MTU || want.DMZEnabled != cur.DMZEnabled ||
		want.DMZIP.String() != cur.DMZIP.String() || want.RespondToPing != cur.RespondToPing || want.RespondToPing6 != cur.RespondToPing6 {
		if err := r.client.SetWANConfig(ctx, want); err != nil {
			return err
		}
	}
	return r.read(ctx, m)
}

func (r *wanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m wanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set WAN", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *wanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m wanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read WAN", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *wanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m wanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set WAN", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *wanResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_wan")
}

func (r *wanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	singletonImport(ctx, req, resp)
}

// ================= skyhub_lan =================

var (
	_ resource.ResourceWithConfigure   = &lanResource{}
	_ resource.ResourceWithImportState = &lanResource{}
)

type lanResource struct{ configured }

type lanModel struct {
	ID            types.String `tfsdk:"id"`
	IP            types.String `tfsdk:"ip"`
	Netmask       types.String `tfsdk:"netmask"`
	DHCPEnabled   types.Bool   `tfsdk:"dhcp_enabled"`
	DHCPPoolStart types.String `tfsdk:"dhcp_pool_start"`
	DHCPPoolEnd   types.String `tfsdk:"dhcp_pool_end"`
	LeaseHours    types.Int64  `tfsdk:"lease_hours"`
}

func newLANResource() resource.Resource { return &lanResource{} }

func (r *lanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan"
}

func (r *lanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "LAN IP, netmask and DHCP pool (singleton, id `default`). **Changing the LAN IP, netmask or " +
			"the DHCP on/off flag restarts the hub** and drops all connections; the provider endpoint must then be updated. " +
			"Destroying only removes the resource from state.",
		Attributes: map[string]schema.Attribute{
			"id":              singletonIDAttribute(),
			"ip":              schema.StringAttribute{MarkdownDescription: "LAN gateway IPv4 address.", Required: true, Validators: []validator.String{ipv4Validator{}}},
			"netmask":         schema.StringAttribute{MarkdownDescription: "LAN netmask.", Required: true, Validators: []validator.String{ipv4Validator{}}},
			"dhcp_enabled":    schema.BoolAttribute{MarkdownDescription: "Run the DHCP server (default true).", Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"dhcp_pool_start": schema.StringAttribute{MarkdownDescription: "First address of the DHCP pool.", Required: true, Validators: []validator.String{ipv4Validator{}}},
			"dhcp_pool_end":   schema.StringAttribute{MarkdownDescription: "Last address of the DHCP pool.", Required: true, Validators: []validator.String{ipv4Validator{}}},
			"lease_hours": schema.Int64Attribute{MarkdownDescription: "Lease time in hours (read-only).", Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *lanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *lanResource) read(ctx context.Context, m *lanModel) error {
	l, err := r.client.LANConfig(ctx)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(singletonID)
	m.IP = types.StringValue(l.IP.String())
	m.Netmask = types.StringValue(l.Netmask.String())
	m.DHCPEnabled = types.BoolValue(l.DHCPEnabled)
	m.DHCPPoolStart = types.StringValue(l.PoolStart.String())
	m.DHCPPoolEnd = types.StringValue(l.PoolEnd.String())
	m.LeaseHours = types.Int64Value(int64(l.LeaseHours))
	return nil
}

func (r *lanResource) apply(ctx context.Context, m *lanModel) error {
	cur, err := r.client.LANConfig(ctx)
	if err != nil {
		return err
	}
	want := skyhub.LANConfig{
		IP: addr(m.IP), Netmask: addr(m.Netmask), DHCPEnabled: boolOr(m.DHCPEnabled, true),
		PoolStart: addr(m.DHCPPoolStart), PoolEnd: addr(m.DHCPPoolEnd),
	}
	if want.IP.String() != cur.IP.String() || want.Netmask.String() != cur.Netmask.String() || want.DHCPEnabled != cur.DHCPEnabled ||
		want.PoolStart.String() != cur.PoolStart.String() || want.PoolEnd.String() != cur.PoolEnd.String() {
		if err := r.client.SetLANConfig(ctx, want); err != nil {
			if errors.Is(err, skyhub.ErrHubRestarting) {
				// The hub is moving to its new address: keep the planned
				// values, do not read back at the old one.
				m.ID = types.StringValue(singletonID)
				m.DHCPEnabled = types.BoolValue(want.DHCPEnabled)
				if m.LeaseHours.IsUnknown() {
					m.LeaseHours = types.Int64Null()
				}
				return errLANRestarting
			}
			return err
		}
	}
	return r.read(ctx, m)
}

var errLANRestarting = errors.New("hub restarting onto its new LAN address")

// lanRestartWarning explains what to do after a LAN change restarted the hub.
func lanRestartWarning(ip string) (string, string) {
	return "Sky Hub restarting onto " + ip,
		"The LAN change was submitted and the hub is restarting. Renew this machine's DHCP lease, then run again with the endpoint pointing at http://" + ip + "/ to apply the remaining resources."
}

func (r *lanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m lanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		if !errors.Is(err, errLANRestarting) {
			resp.Diagnostics.AddError("Cannot set LAN", err.Error())
			return
		}
		resp.Diagnostics.AddWarning(lanRestartWarning(m.IP.ValueString()))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *lanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m lanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read LAN", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *lanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m lanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		if !errors.Is(err, errLANRestarting) {
			resp.Diagnostics.AddError("Cannot set LAN", err.Error())
			return
		}
		resp.Diagnostics.AddWarning(lanRestartWarning(m.IP.ValueString()))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *lanResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_lan")
}

func (r *lanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	singletonImport(ctx, req, resp)
}
