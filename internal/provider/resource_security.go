package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

// ================= skyhub_firewall_globals =================

var (
	_ resource.ResourceWithConfigure   = &firewallGlobalsResource{}
	_ resource.ResourceWithImportState = &firewallGlobalsResource{}
)

type firewallGlobalsResource struct{ configured }

type firewallGlobalsModel struct {
	ID                     types.String `tfsdk:"id"`
	IPv6Firewall           types.Bool   `tfsdk:"ipv6_firewall"`
	IPSecPassthrough       types.Bool   `tfsdk:"ipsec_passthrough"`
	AllowInboundICMPv6Echo types.Bool   `tfsdk:"allow_inbound_icmpv6_echo"`
}

func newFirewallGlobalsResource() resource.Resource { return &firewallGlobalsResource{} }

func (r *firewallGlobalsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_globals"
}

func (r *firewallGlobalsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firewall master toggles on the Firewall Rules page (singleton, id `default`). Rules are managed with `skyhub_firewall_rule`.",
		Attributes: map[string]schema.Attribute{
			"id":                        singletonIDAttribute(),
			"ipv6_firewall":             schema.BoolAttribute{MarkdownDescription: "Enable the IPv6 firewall.", Required: true},
			"ipsec_passthrough":         schema.BoolAttribute{MarkdownDescription: "IPsec passthrough.", Required: true},
			"allow_inbound_icmpv6_echo": schema.BoolAttribute{MarkdownDescription: "Answer inbound ICMPv6 echo requests.", Required: true},
		},
	}
}

func (r *firewallGlobalsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *firewallGlobalsResource) read(ctx context.Context, m *firewallGlobalsModel) error {
	cfg, err := r.client.FirewallConfig(ctx)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(singletonID)
	m.IPv6Firewall = types.BoolValue(cfg.Globals.IPv6Firewall)
	m.IPSecPassthrough = types.BoolValue(cfg.Globals.IPSecPassthrough)
	m.AllowInboundICMPv6Echo = types.BoolValue(cfg.Globals.AllowInboundICMPv6Echo)
	return nil
}

func (r *firewallGlobalsResource) apply(ctx context.Context, m *firewallGlobalsModel) error {
	cfg, err := r.client.FirewallConfig(ctx)
	if err != nil {
		return err
	}
	want := skyhub.FirewallGlobals{
		IPv6Firewall:           m.IPv6Firewall.ValueBool(),
		IPSecPassthrough:       m.IPSecPassthrough.ValueBool(),
		AllowInboundICMPv6Echo: m.AllowInboundICMPv6Echo.ValueBool(),
	}
	if want != cfg.Globals {
		if err := r.client.SetFirewallGlobals(ctx, want); err != nil {
			return err
		}
	}
	return r.read(ctx, m)
}

func (r *firewallGlobalsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m firewallGlobalsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set firewall globals", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *firewallGlobalsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m firewallGlobalsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read firewall globals", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *firewallGlobalsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m firewallGlobalsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot set firewall globals", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *firewallGlobalsResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_firewall_globals")
}

func (r *firewallGlobalsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	singletonImport(ctx, req, resp)
}

// ================= skyhub_wireless =================

var (
	_ resource.ResourceWithConfigure   = &wirelessResource{}
	_ resource.ResourceWithImportState = &wirelessResource{}
)

type wirelessResource struct{ configured }

type wirelessModel struct {
	ID           types.String `tfsdk:"id"`
	Band         types.String `tfsdk:"band"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	SSID         types.String `tfsdk:"ssid"`
	Hidden       types.Bool   `tfsdk:"hidden"`
	Channel      types.Int64  `tfsdk:"channel"`
	Bandwidth    types.String `tfsdk:"bandwidth"`
	PSKWO        types.String `tfsdk:"psk_wo"`
	PSKWOVersion types.Int64  `tfsdk:"psk_wo_version"`
	PSKSet       types.Bool   `tfsdk:"psk_set"`
}

func newWirelessResource() resource.Resource { return &wirelessResource{} }

func (r *wirelessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireless"
}

func (r *wirelessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "WiFi settings of one band (id = band). Security is always WPA2-PSK/AES. " +
			"Saving drops WiFi clients on that band for a few seconds. Isolation, WPS and the " +
			"2.4/5 GHz sync flag are left as they are.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Same as `band`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"band": schema.StringAttribute{
				MarkdownDescription: "`2.4` or `5`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("2.4", "5")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{MarkdownDescription: "Radio on.", Required: true},
			"ssid": schema.StringAttribute{
				MarkdownDescription: "Network name (1-32 characters).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 32)},
			},
			"hidden":  schema.BoolAttribute{MarkdownDescription: "Do not broadcast the SSID.", Optional: true, Computed: true},
			"channel": schema.Int64Attribute{MarkdownDescription: "2.4 GHz: 0 (auto) to 13. 5 GHz: 36 or 44.", Optional: true, Computed: true},
			"bandwidth": schema.StringAttribute{
				MarkdownDescription: "2.4 GHz: `20` or `20/40`. 5 GHz: `80` (channel 36 only) or `40`.",
				Optional:            true, Computed: true,
				Validators: []validator.String{stringvalidator.OneOf("20", "20/40", "40", "80")},
			},
			"psk_wo": schema.StringAttribute{
				MarkdownDescription: "WPA2 key (8-63 characters). Write-only: never stored in state. Sent on create and whenever `psk_wo_version` changes.",
				Optional:            true, Sensitive: true, WriteOnly: true,
				Validators: []validator.String{stringvalidator.LengthBetween(8, 63)},
			},
			"psk_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Change this value to send `psk_wo` again (e.g. the Vault secret's version).",
				Optional:            true,
			},
			"psk_set": schema.BoolAttribute{MarkdownDescription: "The hub has a WPA key configured.", Computed: true},
		},
	}
}

func (r *wirelessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *wirelessResource) radio(ctx context.Context, band string) (*skyhub.WirelessRadio, error) {
	radios, err := r.client.WirelessRadios(ctx)
	if err != nil {
		return nil, err
	}
	for i := range radios {
		if radios[i].Band == band {
			return &radios[i], nil
		}
	}
	return nil, fmt.Errorf("band %s not reported by the hub", band)
}

func (r *wirelessResource) read(ctx context.Context, m *wirelessModel) error {
	band := strOr(m.Band, strOr(m.ID, ""))
	w, err := r.radio(ctx, band)
	if err != nil {
		return err
	}
	m.ID = types.StringValue(band)
	m.Band = types.StringValue(band)
	m.Enabled = types.BoolValue(w.Enabled)
	m.SSID = types.StringValue(w.SSID)
	m.Hidden = types.BoolValue(w.Hidden)
	m.Channel = types.Int64Value(int64(w.Channel))
	m.Bandwidth = types.StringValue(w.Bandwidth)
	m.PSKSet = types.BoolValue(w.PSKSet)
	m.PSKWO = types.StringNull()
	return nil
}

// apply writes the band when anything differs or when psk is non-empty.
func (r *wirelessResource) apply(ctx context.Context, m *wirelessModel, psk string) error {
	band := m.Band.ValueString()
	cur, err := r.radio(ctx, band)
	if err != nil {
		return err
	}
	want := skyhub.WirelessSettings{
		Band:      band,
		Enabled:   m.Enabled.ValueBool(),
		SSID:      m.SSID.ValueString(),
		PSK:       psk,
		Hidden:    boolOr(m.Hidden, cur.Hidden),
		Channel:   int(int64Or(m.Channel, int64(cur.Channel))),
		Bandwidth: strOr(m.Bandwidth, cur.Bandwidth),
	}
	same := want.Enabled == cur.Enabled && want.SSID == cur.SSID && want.Hidden == cur.Hidden &&
		want.Channel == cur.Channel && want.Bandwidth == cur.Bandwidth && psk == ""
	if !same {
		if psk == "" && !cur.PSKSet {
			return fmt.Errorf("the hub has no WPA key on band %s: set psk_wo", band)
		}
		if err := r.client.SetWireless(ctx, want); err != nil {
			return err
		}
	}
	return r.read(ctx, m)
}

func (r *wirelessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m wirelessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	var psk types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("psk_wo"), &psk)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &m, strOr(psk, "")); err != nil {
		resp.Diagnostics.AddError("Cannot set WiFi", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *wirelessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m wirelessModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.read(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot read WiFi", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *wirelessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state wirelessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	var psk types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("psk_wo"), &psk)...)
	if resp.Diagnostics.HasError() {
		return
	}
	send := ""
	if !plan.PSKWOVersion.Equal(state.PSKWOVersion) {
		send = strOr(psk, "")
	}
	if err := r.apply(ctx, &plan, send); err != nil {
		resp.Diagnostics.AddError("Cannot set WiFi", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wirelessResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "skyhub_wireless")
}

func (r *wirelessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "2.4" && req.ID != "5" {
		resp.Diagnostics.AddError("Invalid import id", "Import skyhub_wireless with the band: \"2.4\" or \"5\".")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("band"), types.StringValue(req.ID))...)
}

// ================= skyhub_admin_password =================

var (
	_ resource.ResourceWithConfigure   = &adminPasswordResource{}
	_ resource.ResourceWithImportState = &adminPasswordResource{}
)

type adminPasswordResource struct{ configured }

type adminPasswordModel struct {
	ID                types.String `tfsdk:"id"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64  `tfsdk:"password_wo_version"`
}

func newAdminPasswordResource() resource.Resource { return &adminPasswordResource{} }

func (r *adminPasswordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_password"
}

func (r *adminPasswordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The hub's `admin` password (id `admin`). The provider authenticates with its own `password`; " +
			"when this resource sets a different one, the provider switches to it for the rest of the run. " +
			"Nothing is sent when the target equals the password the provider is using. Destroying only removes it from state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Always `admin`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"password_wo": schema.StringAttribute{
				MarkdownDescription: "New admin password (1-16 characters from the hub's allowed set). Write-only: never stored in state.",
				Required:            true, Sensitive: true, WriteOnly: true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 16)},
			},
			"password_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Change this value to apply `password_wo` again (e.g. the Vault secret's version).",
				Optional:            true,
			},
		},
	}
}

func (r *adminPasswordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *adminPasswordResource) change(ctx context.Context, pw types.String) error {
	if pw.IsNull() || pw.IsUnknown() {
		return fmt.Errorf("password_wo is not known")
	}
	return r.client.ChangeAdminPassword(ctx, pw.ValueString())
}

func (r *adminPasswordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m adminPasswordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	var pw types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &pw)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.change(ctx, pw); err != nil {
		resp.Diagnostics.AddError("Cannot set the admin password", err.Error())
		return
	}
	m.ID = types.StringValue("admin")
	m.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *adminPasswordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// The password cannot be read back; keep the state as is.
	var m adminPasswordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *adminPasswordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adminPasswordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	var pw types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &pw)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.PasswordWOVersion.Equal(state.PasswordWOVersion) {
		if err := r.change(ctx, pw); err != nil {
			resp.Diagnostics.AddError("Cannot set the admin password", err.Error())
			return
		}
	}
	plan.ID = types.StringValue("admin")
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adminPasswordResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Resource removed from state only", "Destroying skyhub_admin_password leaves the hub's password as it is.")
}

func (r *adminPasswordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "admin" {
		resp.Diagnostics.AddError("Invalid import id", "Import skyhub_admin_password with id \"admin\".")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue("admin"))...)
}
