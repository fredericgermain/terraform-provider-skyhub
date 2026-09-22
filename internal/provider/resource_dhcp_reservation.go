package provider

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

var (
	_ resource.Resource                = &dhcpReservationResource{}
	_ resource.ResourceWithConfigure   = &dhcpReservationResource{}
	_ resource.ResourceWithImportState = &dhcpReservationResource{}
)

type dhcpReservationResource struct{ configured }

type dhcpReservationModel struct {
	ID    types.String `tfsdk:"id"`
	MAC   types.String `tfsdk:"mac"`
	IP    types.String `tfsdk:"ip"`
	Name  types.String `tfsdk:"name"`
	Index types.Int64  `tfsdk:"index"`
}

func newDHCPReservationResource() resource.Resource { return &dhcpReservationResource{} }

func (r *dhcpReservationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp_reservation"
}

func (r *dhcpReservationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A DHCP address reservation (LAN IP Setup > Address Reservation). " +
			"Takes effect at the device's next lease renewal; the hub is never rebooted.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The MAC address (lower-case, colon separated).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "Client MAC address, e.g. `aa:bb:cc:dd:ee:ff`. Changing it replaces the reservation.",
				Required:            true,
				Validators:          []validator.String{macValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "Reserved IPv4 address.",
				Required:            true,
				Validators:          []validator.String{ipv4Validator{}},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Device label shown in the hub UI (up to 17 characters).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(17)},
			},
			"index": schema.Int64Attribute{
				MarkdownDescription: "1-based row index on the hub (informational).",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *dhcpReservationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func normMAC(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func (r *dhcpReservationResource) find(ctx context.Context, mac string) (*skyhub.DHCPReservation, error) {
	list, err := r.client.DHCPReservations(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].MAC.String() == normMAC(mac) {
			return &list[i], nil
		}
	}
	return nil, nil
}

func (r *dhcpReservationResource) add(ctx context.Context, m *dhcpReservationModel) error {
	hw, err := net.ParseMAC(m.MAC.ValueString())
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(m.IP.ValueString())
	if err != nil {
		return err
	}
	return r.client.AddDHCPReservation(ctx, skyhub.DHCPReservation{
		MAC: skyhub.MAC{HardwareAddr: hw}, IP: skyhub.Addr{Addr: ip}, Name: m.Name.ValueString(),
	})
}

func (r *dhcpReservationResource) refresh(ctx context.Context, m *dhcpReservationModel) (bool, error) {
	got, err := r.find(ctx, m.MAC.ValueString())
	if err != nil || got == nil {
		return false, err
	}
	m.ID = types.StringValue(got.MAC.String())
	m.MAC = types.StringValue(got.MAC.String())
	m.IP = types.StringValue(got.IP.String())
	m.Name = types.StringValue(got.Name)
	m.Index = types.Int64Value(int64(got.Index))
	return true, nil
}

func (r *dhcpReservationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m dhcpReservationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m.MAC = types.StringValue(normMAC(m.MAC.ValueString()))
	if existing, err := r.find(ctx, m.MAC.ValueString()); err != nil {
		resp.Diagnostics.AddError("Cannot read reservations", err.Error())
		return
	} else if existing != nil {
		if existing.IP.String() != m.IP.ValueString() || existing.Name != m.Name.ValueString() {
			resp.Diagnostics.AddError("Reservation already exists",
				fmt.Sprintf("A reservation for %s already exists on the hub (%s, %q). Import it instead.", m.MAC.ValueString(), existing.IP, existing.Name))
			return
		}
		// Identical entry already on the hub: adopt it.
	} else if err := r.add(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot add reservation", err.Error())
		return
	}
	found, err := r.refresh(ctx, &m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read back reservation", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Reservation not found after creation", "The hub accepted the form but the entry is not listed.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *dhcpReservationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m dhcpReservationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := r.refresh(ctx, &m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read reservation", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *dhcpReservationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dhcpReservationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hw, _ := net.ParseMAC(state.MAC.ValueString())
	if err := r.client.RemoveDHCPReservation(ctx, hw); err != nil && !errors.Is(err, skyhub.ErrNotFound) {
		resp.Diagnostics.AddError("Cannot remove old reservation", err.Error())
		return
	}
	plan.MAC = types.StringValue(normMAC(plan.MAC.ValueString()))
	if err := r.add(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot re-add reservation", err.Error())
		return
	}
	if _, err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot read back reservation", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpReservationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m dhcpReservationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hw, err := net.ParseMAC(m.MAC.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Bad MAC in state", err.Error())
		return
	}
	if err := r.client.RemoveDHCPReservation(ctx, hw); err != nil && !errors.Is(err, skyhub.ErrNotFound) {
		resp.Diagnostics.AddError("Cannot remove reservation", err.Error())
	}
}

func (r *dhcpReservationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := net.ParseMAC(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid import id", "Import with the MAC address, e.g. aa:bb:cc:dd:ee:ff.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mac"), types.StringValue(normMAC(req.ID)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(normMAC(req.ID)))...)
}

// ---- validators ----

type macValidator struct{}

func (macValidator) Description(context.Context) string         { return "must be a MAC address" }
func (macValidator) MarkdownDescription(context.Context) string { return "must be a MAC address" }
func (macValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	hw, err := net.ParseMAC(req.ConfigValue.ValueString())
	if err != nil || len(hw) != 6 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid MAC address", "Expected 6 colon-separated hex bytes.")
	}
}

type ipv4Validator struct{}

func (ipv4Validator) Description(context.Context) string         { return "must be an IPv4 address" }
func (ipv4Validator) MarkdownDescription(context.Context) string { return "must be an IPv4 address" }
func (ipv4Validator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	a, err := netip.ParseAddr(req.ConfigValue.ValueString())
	if err != nil || !a.Is4() {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid IPv4 address", fmt.Sprintf("%q is not an IPv4 address.", req.ConfigValue.ValueString()))
	}
}
