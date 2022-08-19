package ably_control

import (
	"context"

	ably_control_go "github.com/ably/ably-control-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceNamespaceType struct{}

// Get Namespace Resource schema
func (r resourceNamespaceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"app_id": {
				Type:        types.StringType,
				Required:    true,
				Description: "The application ID.",
			},
			"id": {
				Type:        types.StringType,
				Required:    true,
				Description: "The namespace or channel name that the channel rule will apply to.",
			},
			"authenticated": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "Require clients to be authenticated to use channels in this namespace.",
			},
			"persisted": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "If true, messages will be stored for 24 hours.",
			},
			"persist_last": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "If true, the last message on each channel will persist for 365 days.",
			},
			"push_enabled": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "If true, publishing messages with a push payload in the extras field is permitted.",
			},
			"tls_only": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "If true, only clients that are connected using TLS will be permitted to subscribe.",
			},
			"expose_timeserial": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "If true, messages received on a channel will contain a unique timeserial that can be referenced by later messages for use with message interactions.",
			},
		},
	}, nil
}

// New resource instance
func (r resourceNamespaceType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceNamespace{
		p: *(p.(*provider)),
	}, nil
}

type resourceNamespace struct {
	p provider
}

// Create a new resource
func (r resourceNamespace) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	// Checks whether the provider and API Client are configured. If they are not, the provider responds with an error.
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply",
		)
		return
	}

	// Gets plan values
	var plan AblyNamespace
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generates an API request body from the plan values
	namespace_values := ably_control_go.Namespace{
		ID:               plan.ID.Value,
		Authenticated:    plan.Authenticated.Value,
		Persisted:        plan.Persisted.Value,
		PersistLast:      plan.PersistLast.Value,
		PushEnabled:      plan.PushEnabled.Value,
		TlsOnly:          plan.TlsOnly.Value,
		ExposeTimeserial: plan.ExposeTimeserial.Value,
	}

	// Creates a new Ably namespace by invoking the CreateNamespace function from the Client Library
	ably_namespace, err := r.p.client.CreateNamespace(plan.AppID.Value, &namespace_values)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Resource",
			"Could not create resource, unexpected error: "+err.Error(),
		)
		return
	}

	// Maps response body to resource schema attributes.
	resp_apps := AblyNamespace{
		AppID:            types.String{Value: plan.AppID.Value},
		ID:               types.String{Value: ably_namespace.ID},
		Authenticated:    types.Bool{Value: ably_namespace.Authenticated},
		Persisted:        types.Bool{Value: ably_namespace.Persisted},
		PersistLast:      types.Bool{Value: ably_namespace.PersistLast},
		PushEnabled:      types.Bool{Value: ably_namespace.PushEnabled},
		TlsOnly:          types.Bool{Value: ably_namespace.TlsOnly},
		ExposeTimeserial: types.Bool{Value: namespace_values.ExposeTimeserial},
	}

	// Sets state for the new Ably App.
	diags = resp.State.Set(ctx, resp_apps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource
func (r resourceNamespace) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Gets the current state. If it is unable to, the provider responds with an error.
	var state AblyNamespace
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Gets the Ably App ID and namespace ID value for the resource
	app_id := state.AppID.Value
	namespace_id := state.ID.Value

	// Fetches all Ably Namespaces in the app. The function invokes the Client Library Namespaces() method.
	// NOTE: Control API & Client Lib do not currently support fetching single namespace given namespace id
	namespaces, err := r.p.client.Namespaces(app_id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Resource",
			"Could not update resource, unexpected error: "+err.Error(),
		)
		return
	}

	// Loops through namespaces and if id matches, sets state.
	for _, v := range namespaces {
		if v.ID == namespace_id {
			resp_namespaces := AblyNamespace{
				AppID:            types.String{Value: app_id},
				ID:               types.String{Value: namespace_id},
				Authenticated:    types.Bool{Value: v.Authenticated},
				Persisted:        types.Bool{Value: v.Persisted},
				PersistLast:      types.Bool{Value: v.PersistLast},
				PushEnabled:      types.Bool{Value: v.PushEnabled},
				TlsOnly:          types.Bool{Value: v.TlsOnly},
				ExposeTimeserial: types.Bool{Value: v.ExposeTimeserial},
			}
			// Sets state to namespace values.
			diags = resp.State.Set(ctx, &resp_namespaces)

			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}
}

// Update resource
func (r resourceNamespace) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan AblyNamespace
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state AblyNamespace
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Gets the app ID and ID
	app_id := state.AppID.Value
	namespace_id := state.ID.Value

	// Instantiates struct of type ably_control_go.Namespace and sets values to output of plan
	namespace_values := ably_control_go.Namespace{
		ID:               namespace_id,
		Authenticated:    plan.Authenticated.Value,
		Persisted:        plan.Persisted.Value,
		PersistLast:      plan.PersistLast.Value,
		PushEnabled:      plan.PushEnabled.Value,
		TlsOnly:          plan.TlsOnly.Value,
		ExposeTimeserial: plan.ExposeTimeserial.Value,
	}

	// Updates an Ably Namespace. The function invokes the Client Library UpdateNamespace method.
	ably_namespace, err := r.p.client.UpdateNamespace(app_id, &namespace_values)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Resource",
			"Could not update resource, unexpected error: "+err.Error(),
		)
		return
	}

	resp_namespaces := AblyNamespace{
		AppID:            types.String{Value: app_id},
		ID:               types.String{Value: ably_namespace.ID},
		Authenticated:    types.Bool{Value: ably_namespace.Authenticated},
		Persisted:        types.Bool{Value: ably_namespace.Persisted},
		PersistLast:      types.Bool{Value: ably_namespace.PersistLast},
		PushEnabled:      types.Bool{Value: ably_namespace.PushEnabled},
		TlsOnly:          types.Bool{Value: ably_namespace.TlsOnly},
		ExposeTimeserial: types.Bool{Value: ably_namespace.ExposeTimeserial},
	}

	// Sets state to new namespace.
	diags = resp.State.Set(ctx, resp_namespaces)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceNamespace) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	// Get current state
	var state AblyNamespace
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Gets the current state. If it is unable to, the provider responds with an error.
	app_id := state.AppID.Value
	namespace_id := state.ID.Value

	err := r.p.client.DeleteNamespace(app_id, namespace_id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Resource",
			"Could not delete resource, unexpected error: "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

// Import resource
func (r resourceNamespace) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	// Save the import identifier in the id attribute
	// Recent PR in TF Plugin Framework for paths but Hashicorp examples not updated - https://github.com/hashicorp/terraform-plugin-framework/pull/390
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
