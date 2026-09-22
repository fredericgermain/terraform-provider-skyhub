package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fredericgermain/skyhub/pkg/skyhub"
)

// singletonID is the fixed id of resources that model one hub-wide setting.
const singletonID = "default"

// singletonIDAttribute is the shared "id" attribute of singleton resources.
func singletonIDAttribute() schema.Attribute {
	return schema.StringAttribute{
		MarkdownDescription: "Always `default`.",
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// singletonImport accepts only the id "default".
func singletonImport(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != singletonID {
		resp.Diagnostics.AddError("Invalid import id", "This resource is a singleton; import it with id \"default\".")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(singletonID))...)
}

// singletonDeleteWarning removes the resource from state without touching
// the hub.
func singletonDeleteWarning(resp *resource.DeleteResponse, what string) {
	resp.Diagnostics.AddWarning("Resource removed from state only",
		"Destroying "+what+" does not change the hub; the current settings stay in place.")
}

// configured stores the client on a resource.
type configured struct {
	client *skyhub.Client
}

func (c *configured) configure(req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := clientFrom(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "expected *skyhub.Client")
		return
	}
	c.client = client
}

// strOr returns the state string or def when null/unknown.
func strOr(v types.String, def string) string {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueString()
}

func boolOr(v types.Bool, def bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueBool()
}

func int64Or(v types.Int64, def int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueInt64()
}
