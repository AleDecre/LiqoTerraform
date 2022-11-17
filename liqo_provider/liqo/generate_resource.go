package liqo

import (
	"context"
	"terraform-provider-test/liqo/attribute_plan_modifier"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/liqotech/liqo/pkg/auth"
	"github.com/liqotech/liqo/pkg/utils"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &generateResource{}
	_ resource.ResourceWithConfigure = &generateResource{}
)

// NewGenerateResource is a helper function to simplify the provider implementation.
func NewGenerateResource() resource.Resource {
	return &generateResource{}
}

// generateResource is the resource implementation.
type generateResource struct {
	kubeconfig kubeconfig
}

// Metadata returns the resource type name.
func (r *generateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generate"
}

// GetSchema defines the schema for the resource.
func (r *generateResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"cluster_id": {
				Type:     types.StringType,
				Computed: true,
			},
			"cluster_name": {
				Type:     types.StringType,
				Computed: true,
			},
			"auth_ep": {
				Type:     types.StringType,
				Computed: true,
			},
			"local_token": {
				Type:     types.StringType,
				Computed: true,
			},
			"liqo_namespace": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					attribute_plan_modifier.DefaultValue(types.StringValue("liqo")),
				},
				Computed: true,
			},
			"error_msg": {
				Type:     types.StringType,
				Computed: true,
			},
		},
	}, nil
}

// Create a new resource
func (r *generateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan generateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ErrorMsg = types.StringValue("Success")

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.Value)
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}
	_ = clusterIdentity

	localToken, err := auth.GetToken(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.Value)
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	authEP, err := foreigncluster.GetHomeAuthURL(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.Value)
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	if clusterIdentity.ClusterName == "" {
		clusterIdentity.ClusterName = clusterIdentity.ClusterID
	}

	plan.ClusterID = types.StringValue(clusterIdentity.ClusterID)
	plan.ClusterName = types.StringValue(clusterIdentity.ClusterName)
	plan.LocalToken = types.StringValue(localToken)
	plan.AuthEP = types.StringValue(authEP)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r *generateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state generateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *generateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan generateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ErrorMsg = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *generateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// Configure adds the provider configured client to the resource.
func (r *generateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.kubeconfig = req.ProviderData.(kubeconfig)
}

// generateResourceModel maps the resource schema data.
type generateResourceModel struct {
	ClusterID     types.String `tfsdk:"cluster_id"`
	ClusterName   types.String `tfsdk:"cluster_name"`
	AuthEP        types.String `tfsdk:"auth_ep"`
	LocalToken    types.String `tfsdk:"local_token"`
	LiqoNamespace types.String `tfsdk:"liqo_namespace"`
	ErrorMsg      types.String `tfsdk:"error_msg"`
}
