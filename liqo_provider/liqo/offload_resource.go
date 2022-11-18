package liqo

import (
	"context"
	"terraform-provider-test/liqo/attribute_plan_modifier"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	offloadingv1alpha1 "github.com/liqotech/liqo/apis/offloading/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &offloadResource{}
	_ resource.ResourceWithConfigure = &offloadResource{}
)

// NewOffloadResource is a helper function to simplify the provider implementation.
func NewOffloadResource() resource.Resource {
	return &offloadResource{}
}

// offloadResource is the resource implementation.
type offloadResource struct {
	kubeconfig kubeconfig
}

// Metadata returns the resource type name.
func (o *offloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_offload"
}

// GetSchema defines the schema for the resource.
func (o *offloadResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"namespace": {
				Type:     types.StringType,
				Required: true,
			},
			"pod_offloading_strategy": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					attribute_plan_modifier.DefaultValue(types.StringValue("LocalAndRemote")),
				},
				Computed: true,
			},
			"namespace_mapping_strategy": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					attribute_plan_modifier.DefaultValue(types.StringValue("DefaultName")),
				},
				Computed: true,
			},
			"cluster_selector": {
				Type:     types.ListType{ElemType: types.StringType},
				Optional: true,
			},
		},
	}, nil
}

// Create a new resource
func (o *offloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan offloadResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ClusterSelector [][]metav1.LabelSelectorRequirement

	for _, selector := range plan.ClusterSelector {
		s, err := metav1.ParseToLabelSelector(selector.Value)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Resource",
				err.Error(),
			)
			return
		}

		// Convert MatchLabels into MatchExpressions
		for key, value := range s.MatchLabels {
			req := metav1.LabelSelectorRequirement{Key: key, Operator: metav1.LabelSelectorOpIn, Values: []string{value}}
			s.MatchExpressions = append(s.MatchExpressions, req)
		}

		ClusterSelector = append(ClusterSelector, s.MatchExpressions)
	}

	terms := []corev1.NodeSelectorTerm{}

	for _, selector := range ClusterSelector {
		var requirements []corev1.NodeSelectorRequirement

		for _, r := range selector {
			requirements = append(requirements, corev1.NodeSelectorRequirement{
				Key:      r.Key,
				Operator: corev1.NodeSelectorOperator(r.Operator),
				Values:   r.Values,
			})
		}

		terms = append(terms, corev1.NodeSelectorTerm{MatchExpressions: requirements})
	}

	nsoff := &offloadingv1alpha1.NamespaceOffloading{ObjectMeta: metav1.ObjectMeta{
		Name: consts.DefaultNamespaceOffloadingName, Namespace: plan.Namespace.Value}}

	//var oldStrategy offloadingv1alpha1.PodOffloadingStrategyType
	_, err := controllerutil.CreateOrUpdate(ctx, o.kubeconfig.CRClient, nsoff, func() error {
		//oldStrategy = nsoff.Spec.PodOffloadingStrategy
		nsoff.Spec.PodOffloadingStrategy = offloadingv1alpha1.PodOffloadingStrategyType(plan.PodOffloadingStrategy.Value)
		nsoff.Spec.NamespaceMappingStrategy = offloadingv1alpha1.NamespaceMappingStrategyType(plan.NamespaceMappingStrategy.Value)
		nsoff.Spec.ClusterSelector = corev1.NodeSelector{NodeSelectorTerms: terms}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)

		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (o *offloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state offloadResourceModel
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
func (o *offloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (o *offloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var data offloadResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	nsoff := &offloadingv1alpha1.NamespaceOffloading{ObjectMeta: metav1.ObjectMeta{
		Name: consts.DefaultNamespaceOffloadingName, Namespace: data.Namespace.Value}}
	if err := o.kubeconfig.CRClient.Delete(ctx, nsoff); client.IgnoreNotFound(err) != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			err.Error(),
		)
		return
	}

}

// Configure adds the provider configured client to the resource.
func (o *offloadResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	o.kubeconfig = req.ProviderData.(kubeconfig)
}

// offloadResourceModel maps the resource schema data.
type offloadResourceModel struct {
	Namespace                types.String   `tfsdk:"namespace"`
	PodOffloadingStrategy    types.String   `tfsdk:"pod_offloading_strategy"`
	NamespaceMappingStrategy types.String   `tfsdk:"namespace_mapping_strategy"`
	ClusterSelector          []types.String `tfsdk:"cluster_selector"`
}
