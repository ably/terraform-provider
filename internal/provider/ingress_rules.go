package ably_control

import (
	"context"
	"fmt"
	"strings"

	ably_control_go "github.com/ably/ably-control-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfsdk_resource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// converts ingress rule from terraform format to control sdk format
func GetPlanIngressRule(plan AblyIngressRule) ably_control_go.NewIngressRule {
	var target ably_control_go.IngressTarget

	switch t := plan.Target.(type) {
	case *AblyIngressRuleTargetMongo:
		target = &ably_control_go.IngressMongoTarget{
			Url:                      t.Url,
			Database:                 t.Database,
			Collection:               t.Collection,
			Pipeline:                 t.Pipeline,
			FullDocument:             t.FullDocument,
			FullDocumentBeforeChange: t.FullDocumentBeforeChange,
			PrimarySite:              t.PrimarySite,
		}
	case *AblyIngressRuleTargetPostgresOutbox:
		target = &ably_control_go.IngressPostgresOutboxTarget{
			Url:               t.Url,
			OutboxTableSchema: t.OutboxTableSchema,
			OutboxTableName:   t.OutboxTableName,
			NodesTableSchema:  t.NodesTableSchema,
			NodesTableName:    t.NodesTableName,
			SslMode:           t.SslMode,
			SslRootCert:       t.SslRootCert,
			PrimarySite:       t.PrimarySite,
		}
	}

	rule_values := ably_control_go.NewIngressRule{
		Status: plan.Status.ValueString(),
		Target: target,
	}

	return rule_values
}

// Maps response body to resource schema attributes.
// Using plan to fill in values that the api does not return.
func GetIngressRuleResponse(ably_ingress_rule *ably_control_go.IngressRule, plan *AblyIngressRule) AblyIngressRule {
	var resp_target interface{}

	switch v := ably_ingress_rule.Target.(type) {
	case *ably_control_go.IngressMongoTarget:
		resp_target = &AblyIngressRuleTargetMongo{
			Url:                      v.Url,
			Database:                 v.Database,
			Collection:               v.Collection,
			Pipeline:                 v.Pipeline,
			FullDocument:             v.FullDocument,
			FullDocumentBeforeChange: v.FullDocumentBeforeChange,
			PrimarySite:              v.PrimarySite,
		}
	case *ably_control_go.IngressPostgresOutboxTarget:
		resp_target = &AblyIngressRuleTargetPostgresOutbox{
			Url:               v.Url,
			OutboxTableSchema: v.OutboxTableSchema,
			OutboxTableName:   v.OutboxTableName,
			NodesTableSchema:  v.NodesTableSchema,
			NodesTableName:    v.NodesTableName,
			SslMode:           v.SslMode,
			SslRootCert:       v.SslRootCert,
			PrimarySite:       v.PrimarySite,
		}
	}

	resp_rule := AblyIngressRule{
		ID:     types.StringValue(ably_ingress_rule.ID),
		AppID:  types.StringValue(ably_ingress_rule.AppID),
		Status: types.StringValue(ably_ingress_rule.Status),
		Target: resp_target,
	}

	return resp_rule
}

func GetIngressRuleSchema(target map[string]tfsdk.Attribute, markdown_description string) tfsdk.Schema {
	return tfsdk.Schema{
		MarkdownDescription: markdown_description,
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:        types.StringType,
				Computed:    true,
				Description: "The rule ID.",
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk_resource.UseStateForUnknown(),
				},
			},
			"app_id": {
				Type:        types.StringType,
				Required:    true,
				Description: "The Ably application ID.",
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk_resource.RequiresReplace(),
				},
			},
			"status": {
				Type:        types.StringType,
				Optional:    true,
				Description: "The status of the rule. Rules can be enabled or disabled.",
			},
			"target": {
				Required:    true,
				Description: "object (rule_source)",
				Attributes:  tfsdk.SingleNestedAttributes(target),
			},
		},
	}
}

type IngressRule interface {
	Provider() *provider
	Name() string
}

// Create a new resource
func CreateIngressRule[T any](r IngressRule, ctx context.Context, req tfsdk_resource.CreateRequest, resp *tfsdk_resource.CreateResponse) {
	// Checks whether the provider and API Client are configured. If they are not, the provider responds with an error.
	if !r.Provider().configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply",
		)
		return
	}

	// Gets plan values
	var p AblyIngressRuleDecoder[*T]
	diags := req.Plan.Get(ctx, &p)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := p.IngressRule()
	plan_values := GetPlanIngressRule(plan)

	// Creates a new Ably Ingress Rule by invoking the CreateRule function from the Client Library
	ingress_rule, err := r.Provider().client.CreateIngressRule(plan.AppID.ValueString(), &plan_values)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating Resource '%s'", r.Name()),
			fmt.Sprintf("Could not create resource '%s', unexpected error: %s", r.Name(), err.Error()),
		)

		return
	}

	response_values := GetIngressRuleResponse(&ingress_rule, &plan)

	// Sets state for the new Ably Ingress Rule.
	diags = resp.State.Set(ctx, response_values)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource
func ReadIngressRule[T any](r IngressRule, ctx context.Context, req tfsdk_resource.ReadRequest, resp *tfsdk_resource.ReadResponse) {
	// Gets the current state. If it is unable to, the provider responds with an error.
	var s AblyIngressRuleDecoder[*T]
	diags := req.State.Get(ctx, &s)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	state := s.IngressRule()

	// Gets the Ably App ID and Ably Ingress Rule ID value for the resource
	app_id := s.AppID.ValueString()
	ingress_rule_id := s.ID.ValueString()

	// Get Ingress Rule data
	ingress_rule, err := r.Provider().client.IngressRule(app_id, ingress_rule_id)

	if err != nil {
		if is_404(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading Resource %s", r.Name()),
			fmt.Sprintf("Could not read resource %s, unexpected error: %s", r.Name(), err.Error()),
		)
		return
	}

	response_values := GetIngressRuleResponse(&ingress_rule, &state)

	// Sets state to app values.
	diags = resp.State.Set(ctx, &response_values)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

// // Update resource
func UpdateIngressRule[T any](r IngressRule, ctx context.Context, req tfsdk_resource.UpdateRequest, resp *tfsdk_resource.UpdateResponse) {
	// Gets plan values
	var p AblyIngressRuleDecoder[*T]
	diags := req.Plan.Get(ctx, &p)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan := p.IngressRule()

	rule_values := GetPlanIngressRule(plan)

	// Gets the Ably App ID and Ably Ingress Rule ID value for the resource
	app_id := plan.AppID.ValueString()
	rule_id := plan.ID.ValueString()

	// Update Ably Ingress Rule
	ingress_rule, err := r.Provider().client.UpdateIngressRule(app_id, rule_id, &rule_values)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating Resource %s", r.Name()),
			fmt.Sprintf("Could not update resource %s, unexpected error: %s", r.Name(), err.Error()),
		)
		return
	}

	response_values := GetIngressRuleResponse(&ingress_rule, &plan)

	// Sets state to app values.
	diags = resp.State.Set(ctx, &response_values)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func DeleteIngressRule[T any](r IngressRule, ctx context.Context, req tfsdk_resource.DeleteRequest, resp *tfsdk_resource.DeleteResponse) {
	// Gets the current state. If it is unable to, the provider responds with an error.
	var s AblyIngressRuleDecoder[*T]
	diags := req.State.Get(ctx, &s)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	state := s.IngressRule()

	// Gets the Ably App ID and Ably Rule ID value for the resource
	app_id := state.AppID.ValueString()
	ingress_rule_id := state.ID.ValueString()

	err := r.Provider().client.DeleteIngressRule(app_id, ingress_rule_id)
	if err != nil {
		if is_404(err) {
			resp.Diagnostics.AddWarning(
				fmt.Sprintf("Resource does %s not exist", r.Name()),
				fmt.Sprintf("Resource does %s not exist, it may have already been deleted: %s", r.Name(), err.Error()),
			)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error deleting Resource %s'", r.Name()),
				fmt.Sprintf("Could not delete resource '%s', unexpected error: %s", r.Name(), err.Error()),
			)
			return
		}
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

// // Import resource
func ImportIngressRuleResource(ctx context.Context, req tfsdk_resource.ImportStateRequest, resp *tfsdk_resource.ImportStateResponse, fields ...string) {
	// Save the import identifier in the id attribute
	// identifier should be in the format app_id,key_id
	idParts := strings.Split(req.ID, ",")
	anyEmpty := false

	for _, v := range idParts {
		if v == "" {
			anyEmpty = true
		}
	}

	if len(idParts) != len(fields) || anyEmpty {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: '%s'. Got: %q", strings.Join(fields, ","), req.ID),
		)
		return
	}
	// Recent PR in TF Plugin Framework for paths but Hashicorp examples not updated - https://github.com/hashicorp/terraform-plugin-framework/pull/390
	for i, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(v), idParts[i])...)
	}
}
