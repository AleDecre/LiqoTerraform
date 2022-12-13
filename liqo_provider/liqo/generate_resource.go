package liqo

import (
	"context"
	"terraform-provider-test/liqo/attribute_plan_modifier"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/liqotech/liqo/pkg/auth"
	"github.com/liqotech/liqo/pkg/utils"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
)

var (
	_ resource.Resource              = &generateResource{}
	_ resource.ResourceWithConfigure = &generateResource{}
)

func NewGenerateResource() resource.Resource {
	return &generateResource{}
}

type generateResource struct {
	kubeconfig kubeconfig
}

func (r *generateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generate"
}

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
		},
	}, nil
}

// Creation of Generate Resource to obtain necessary pairing parameters used by Peering Resources
// This resource will reproduce the same effect and outputs of "liqoctl generate peer-command" command
func (r *generateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan generateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	localToken, err := auth.GetToken(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	authEP, err := foreigncluster.GetHomeAuthURL(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	if clusterIdentity.ClusterName == "" {
		clusterIdentity.ClusterName = clusterIdentity.ClusterID
	}

	plan.ClusterID = types.StringValue(clusterIdentity.ClusterID)
	plan.ClusterName = types.StringValue(clusterIdentity.ClusterName)
	plan.LocalToken = types.StringValue(localToken)
	plan.AuthEP = types.StringValue(authEP)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *generateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state generateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *generateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unable to Update Resource",
		"Update is not supported/permitted yet.",
	)
}

func (r *generateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// Configure method to obtain kubernetes Clients provided by provider
func (r *generateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.kubeconfig = req.ProviderData.(kubeconfig)
}

type generateResourceModel struct {
	ClusterID     types.String `tfsdk:"cluster_id"`
	ClusterName   types.String `tfsdk:"cluster_name"`
	AuthEP        types.String `tfsdk:"auth_ep"`
	LocalToken    types.String `tfsdk:"local_token"`
	LiqoNamespace types.String `tfsdk:"liqo_namespace"`
}
