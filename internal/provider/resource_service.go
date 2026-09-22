package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{}
)

type serviceResource struct{ configured }

type serviceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Protocol  types.String `tfsdk:"protocol"`
	StartPort types.Int64  `tfsdk:"start_port"`
	EndPort   types.Int64  `tfsdk:"end_port"`
	Index     types.Int64  `tfsdk:"index"`
}

func newServiceResource() resource.Resource { return &serviceResource{} }

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A custom port service (Security > Services) that firewall rules can reference. " +
			"Changing anything but the name recreates the entry on the hub; rules referencing it briefly lose their target.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The service name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service name, 1-12 characters from `A-Z a-z 0-9 - _`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 12),
					stringvalidator.RegexMatches(serviceNameRE, "letters, digits, - and _ only"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "`tcp`, `udp` or `tcp_udp`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("tcp", "udp", "tcp_udp")},
			},
			"start_port": schema.Int64Attribute{
				MarkdownDescription: "First port (1-65535).",
				Required:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 65535)},
			},
			"end_port": schema.Int64Attribute{
				MarkdownDescription: "Last port; defaults to `start_port`.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 65535)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"index": schema.Int64Attribute{
				MarkdownDescription: "1-based row index on the hub (informational).",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req, resp)
}

func (r *serviceResource) find(ctx context.Context, name string) (*skyhub.Service, error) {
	list, err := r.client.Services(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Name == name {
			return &list[i], nil
		}
	}
	return nil, nil
}

func (r *serviceResource) refresh(ctx context.Context, m *serviceModel) (bool, error) {
	got, err := r.find(ctx, m.Name.ValueString())
	if err != nil || got == nil {
		return false, err
	}
	m.ID = types.StringValue(got.Name)
	m.Protocol = types.StringValue(got.Protocol)
	m.StartPort = types.Int64Value(int64(got.StartPort))
	m.EndPort = types.Int64Value(int64(got.EndPort))
	m.Index = types.Int64Value(int64(got.Index))
	return true, nil
}

func (r *serviceResource) add(ctx context.Context, m *serviceModel) error {
	end := int(int64Or(m.EndPort, m.StartPort.ValueInt64()))
	return r.client.AddService(ctx, skyhub.Service{
		Name: m.Name.ValueString(), Protocol: m.Protocol.ValueString(),
		StartPort: int(m.StartPort.ValueInt64()), EndPort: end,
	})
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if existing, err := r.find(ctx, m.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Cannot read services", err.Error())
		return
	} else if existing != nil {
		resp.Diagnostics.AddError("Service already exists", "A service named "+m.Name.ValueString()+" exists on the hub. Import it instead.")
		return
	}
	if err := r.add(ctx, &m); err != nil {
		resp.Diagnostics.AddError("Cannot add service", err.Error())
		return
	}
	found, err := r.refresh(ctx, &m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read back service", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Service not found after creation", "The hub accepted the form but the entry is not listed.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := r.refresh(ctx, &m)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read service", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveService(ctx, state.Name.ValueString()); err != nil && !errors.Is(err, skyhub.ErrNotFound) {
		resp.Diagnostics.AddError("Cannot remove old service", err.Error())
		return
	}
	if err := r.add(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot re-add service", err.Error())
		return
	}
	if _, err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Cannot read back service", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveService(ctx, m.Name.ValueString()); err != nil && !errors.Is(err, skyhub.ErrNotFound) {
		resp.Diagnostics.AddError("Cannot remove service", err.Error())
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}
