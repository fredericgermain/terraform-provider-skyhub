package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

var (
	_ resource.Resource                = &firewallRuleResource{}
	_ resource.ResourceWithConfigure   = &firewallRuleResource{}
	_ resource.ResourceWithImportState = &firewallRuleResource{}
)

type firewallRuleResource struct{ configured }

type firewallRuleModel struct {
	ID           types.String `tfsdk:"id"`
	Direction    types.String `tfsdk:"direction"`
	IPVersion    types.Int64  `tfsdk:"ip_version"`
	Service      types.String `tfsdk:"service"`
	Action       types.String `tfsdk:"action"`
	LANType      types.String `tfsdk:"lan_type"`
	LANStartIP   types.String `tfsdk:"lan_start_ip"`
	LANEndIP     types.String `tfsdk:"lan_end_ip"`
	WANType      types.String `tfsdk:"wan_type"`
	WANStartIP   types.String `tfsdk:"wan_start_ip"`
	WANEndIP     types.String `tfsdk:"wan_end_ip"`
	LANIPv6      types.String `tfsdk:"lan_ipv6"`
	WANStartIPv6 types.String `tfsdk:"wan_start_ipv6"`
	WANEndIPv6   types.String `tfsdk:"wan_end_ipv6"`
	Logging      types.String `tfsdk:"logging"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Position     types.Int64  `tfsdk:"position"`
	LANUsers     types.String `tfsdk:"lan_users"`
	WANServers   types.String `tfsdk:"wan_servers"`
}

func newFirewallRuleResource() resource.Resource { return &firewallRuleResource{} }

func (r *firewallRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule"
}

func (r *firewallRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An inbound or outbound firewall rule (Security > Firewall Rules). " +
			"A rule is identified by its direction, IP version and service name; the hub allows one rule per service and direction. " +
			"Changing `action`, addresses or `logging` deletes and re-adds the rule (it is briefly absent); `enabled` and `position` " +
			"are applied through the list page without recreating it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`<direction>/<ip_version>/<service>`, e.g. `in/4/SSH`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "`in` (WAN to LAN) or `out` (LAN to WAN).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("in", "out")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ip_version": schema.Int64Attribute{
				MarkdownDescription: "4 or 6 (default 4).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(4),
				Validators:          []validator.Int64{int64validator.OneOf(4, 6)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "Built-in service name (e.g. `SSH`, `HTTP`, `Any(All)`) or a `skyhub_service` name.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "`allow_always`, `allow_schedule`, `block_always` or `block_schedule`.",
				Required:            true,
				Validators: []validator.String{stringvalidator.OneOf(
					skyhub.ActionAllowAlways, skyhub.ActionAllowSchedule, skyhub.ActionBlockAlways, skyhub.ActionBlockSchedule)},
			},
			"lan_type": schema.StringAttribute{
				MarkdownDescription: "Outbound IPv4 only: `any`, `single` or `range` (default `any`). Inbound IPv4 rules always target the single host in `lan_start_ip`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("any"),
				Validators:          []validator.String{stringvalidator.OneOf("any", "single", "range")},
			},
			"lan_start_ip": schema.StringAttribute{
				MarkdownDescription: "LAN IPv4 host (inbound: required; outbound: for `single`/`range`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"lan_end_ip": schema.StringAttribute{
				MarkdownDescription: "End of the LAN range (outbound `range`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"wan_type": schema.StringAttribute{
				MarkdownDescription: "`any`, `single` or `range` (default `any`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("any"),
				Validators:          []validator.String{stringvalidator.OneOf("any", "single", "range")},
			},
			"wan_start_ip": schema.StringAttribute{
				MarkdownDescription: "WAN IPv4 address (for `single`/`range`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"wan_end_ip": schema.StringAttribute{
				MarkdownDescription: "End of the WAN range.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"lan_ipv6": schema.StringAttribute{
				MarkdownDescription: "IPv6 rules: `<prefix>|<interface id>` (e.g. `2001:db8:1:2|::10`) or just an interface id to use the hub's global prefix.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"wan_start_ipv6": schema.StringAttribute{
				MarkdownDescription: "IPv6 rules: WAN address for `single`/`range`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"wan_end_ipv6": schema.StringAttribute{
				MarkdownDescription: "IPv6 rules: end of the WAN range.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"logging": schema.StringAttribute{
				MarkdownDescription: "`never`, `always`, `match` or `not_match` (default `never`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(skyhub.LogNever),
				Validators:          []validator.String{stringvalidator.OneOf(skyhub.LogNever, skyhub.LogAlways, skyhub.LogMatch, skyhub.LogNotMatch)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the rule is active (default true).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"position": schema.Int64Attribute{
				MarkdownDescription: "1-based position in the list. Unset: appended at the end and tracked.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"lan_users": schema.StringAttribute{
				MarkdownDescription: "LAN column as displayed by the hub (informational).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wan_servers": schema.StringAttribute{
				MarkdownDescription: "WAN column as displayed by the hub (informational).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *firewallRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func ruleID(dir string, ipv int64, service string) string {
	return fmt.Sprintf("%s/%d/%s", dir, ipv, service)
}

func (m *firewallRuleModel) toRule() skyhub.FirewallRule {
	return skyhub.FirewallRule{
		Direction:  skyhub.Direction(m.Direction.ValueString()),
		IPVersion:  int(int64Or(m.IPVersion, 4)),
		Service:    m.Service.ValueString(),
		Action:     m.Action.ValueString(),
		Logging:    strOr(m.Logging, skyhub.LogNever),
		LANType:    strOr(m.LANType, "any"),
		LANStart:   m.LANStartIP.ValueString(),
		LANEnd:     m.LANEndIP.ValueString(),
		WANType:    strOr(m.WANType, "any"),
		WANStart:   m.WANStartIP.ValueString(),
		WANEnd:     m.WANEndIP.ValueString(),
		LANIPv6:    m.LANIPv6.ValueString(),
		WANIPv6:    m.WANStartIPv6.ValueString(),
		WANIPv6End: m.WANEndIPv6.ValueString(),
	}
}

// refresh fills the computed/hub-visible attributes from the hub. Address
// attributes are not shown separately by the hub, so they keep their
// configured values.
func (r *firewallRuleResource) refresh(ctx context.Context, m *firewallRuleModel) (bool, error) {
	cfg, err := r.client.FirewallConfig(ctx)
	if err != nil {
		return false, err
	}
	got := cfg.Find(skyhub.Direction(m.Direction.ValueString()), int(int64Or(m.IPVersion, 4)), m.Service.ValueString())
	if got == nil {
		return false, nil
	}
	m.ID = types.StringValue(ruleID(m.Direction.ValueString(), int64Or(m.IPVersion, 4), m.Service.ValueString()))
	m.IPVersion = types.Int64Value(int64(got.IPVersion))
	m.Action = types.StringValue(got.Action)
	m.Logging = types.StringValue(got.Logging)
	m.Enabled = types.BoolValue(got.Enabled)
	m.Position = types.Int64Value(int64(got.Position))
	m.LANUsers = types.StringValue(got.LANUsers)
	m.WANServers = types.StringValue(got.WANServers)
	for _, s := range []*types.String{&m.LANType, &m.WANType} {
		if s.IsNull() || s.IsUnknown() {
			*s = types.StringValue("any")
		}
	}
	for _, s := range []*types.String{&m.LANStartIP, &m.LANEndIP, &m.WANStartIP, &m.WANEndIP, &m.LANIPv6, &m.WANStartIPv6, &m.WANEndIPv6} {
		if s.IsNull() || s.IsUnknown() {
			*s = types.StringValue("")
		}
	}
	return true, nil
}

// applyEnabledPosition moves/toggles the rule through the list page.
func (r *firewallRuleResource) applyEnabledPosition(ctx context.Context, dir skyhub.Direction, service string, enabled bool, position int64) error {
	cfg, err := r.client.FirewallConfig(ctx)
	if err != nil {
		return err
	}
	rules := cfg.Rules(dir)
	cur := cfg.Find(dir, 0, service)
	if cur == nil {
		return skyhub.ErrNotFound
	}
	var order []string
	for _, x := range rules {
		if x.Service != service {
			order = append(order, x.Service)
		}
	}
	pos := int(position)
	if pos < 1 {
		pos = cur.Position
	}
	if pos > len(rules) {
		pos = len(rules)
	}
	order = append(order[:pos-1], append([]string{service}, order[pos-1:]...)...)
	same := cur.Enabled == enabled && cur.Position == pos
	if same {
		return nil
	}
	return r.client.SetFirewallRuleSet(ctx, dir, order, map[string]bool{service: enabled})
}

func (r *firewallRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m firewallRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dir := skyhub.Direction(m.Direction.ValueString())
	if err := r.client.AddFirewallRule(ctx, m.toRule()); err != nil {
		resp.Diagnostics.AddError("Cannot add firewall rule", err.Error())
		return
	}
	if !boolOr(m.Enabled, true) || !m.Position.IsNull() && !m.Position.IsUnknown() {
		if err := r.applyEnabledPosition(ctx, dir, m.Service.ValueString(), boolOr(m.Enabled, true), int64Or(m.Position, 0)); err != nil {
			resp.Diagnostics.AddError("Rule added but enabled/position could not be applied", err.Error())
			return
		}
	}
	found, err := r.refresh(ctx, &m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read back rule", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Rule not found after creation", "The hub accepted the form but the rule is not listed.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *firewallRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := r.refresh(ctx, &m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read firewall rule", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *firewallRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state firewallRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dir := skyhub.Direction(plan.Direction.ValueString())
	recreate := plan.Action.ValueString() != state.Action.ValueString() ||
		strOr(plan.Logging, "never") != strOr(state.Logging, "never") ||
		strOr(plan.LANType, "any") != strOr(state.LANType, "any") ||
		plan.LANStartIP.ValueString() != state.LANStartIP.ValueString() ||
		plan.LANEndIP.ValueString() != state.LANEndIP.ValueString() ||
		strOr(plan.WANType, "any") != strOr(state.WANType, "any") ||
		plan.WANStartIP.ValueString() != state.WANStartIP.ValueString() ||
		plan.WANEndIP.ValueString() != state.WANEndIP.ValueString() ||
		plan.LANIPv6.ValueString() != state.LANIPv6.ValueString() ||
		plan.WANStartIPv6.ValueString() != state.WANStartIPv6.ValueString() ||
		plan.WANEndIPv6.ValueString() != state.WANEndIPv6.ValueString()
	position := int64Or(plan.Position, int64Or(state.Position, 0))
	if recreate {
		if err := r.client.RemoveFirewallRule(ctx, dir, int(int64Or(state.IPVersion, 4)), state.Service.ValueString()); err != nil && !errors.Is(err, skyhub.ErrNotFound) {
			resp.Diagnostics.AddError("Cannot remove old rule", err.Error())
			return
		}
		if err := r.client.AddFirewallRule(ctx, plan.toRule()); err != nil {
			resp.Diagnostics.AddError("Cannot re-add rule", err.Error())
			return
		}
	}
	if err := r.applyEnabledPosition(ctx, dir, plan.Service.ValueString(), boolOr(plan.Enabled, true), position); err != nil {
		resp.Diagnostics.AddError("Cannot apply enabled/position", err.Error())
		return
	}
	if _, err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot read back rule", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.RemoveFirewallRule(ctx, skyhub.Direction(m.Direction.ValueString()), int(int64Or(m.IPVersion, 4)), m.Service.ValueString())
	if err != nil && !errors.Is(err, skyhub.ErrNotFound) {
		resp.Diagnostics.AddError("Cannot remove firewall rule", err.Error())
	}
}

func (r *firewallRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || (parts[0] != "in" && parts[0] != "out") {
		resp.Diagnostics.AddError("Invalid import id", "Use <direction>/<ip_version>/<service>, e.g. in/4/SSH.")
		return
	}
	ipv, err := strconv.Atoi(parts[1])
	if err != nil || (ipv != 4 && ipv != 6) {
		resp.Diagnostics.AddError("Invalid import id", "ip_version must be 4 or 6.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("direction"), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip_version"), types.Int64Value(int64(ipv)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), types.StringValue(parts[2]))...)
}
