package liqo

import (
	"context"
	"fmt"
	"terraform-provider-test/liqo/attribute_plan_modifier"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
	"github.com/liqotech/liqo/pkg/discovery"
	"github.com/liqotech/liqo/pkg/utils"
	authenticationtokenutils "github.com/liqotech/liqo/pkg/utils/authenticationtoken"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &peeringResource{}
	_ resource.ResourceWithConfigure = &peeringResource{}
)

// NewPeeringResource is a helper function to simplify the provider implementation.
func NewPeeringResource() resource.Resource {
	return &peeringResource{}
}

// peeringResource is the resource implementation.
type peeringResource struct {
	kubeconfig kubeconfig
}

// Metadata returns the resource type name.
func (r *peeringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peering"
}

// GetSchema defines the schema for the resource.
func (r *peeringResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"cluster_id": {
				Type:     types.StringType,
				Required: true,
			},
			"cluster_name": {
				Type:     types.StringType,
				Required: true,
			},
			"cluster_authurl": {
				Type:     types.StringType,
				Required: true,
			},
			"cluster_token": {
				Type:     types.StringType,
				Required: true,
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

// Create a new resource
func (r *peeringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan peeringResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, r.kubeconfig.CRClient, plan.LiqoNamespace.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	if clusterIdentity.ClusterID == plan.ClusterID.Value {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			"Same ClusterID",
		)
		return
	}

	err = authenticationtokenutils.StoreInSecret(ctx, r.kubeconfig.KubeClient, plan.ClusterID.Value, plan.ClusterToken.Value, plan.LiqoNamespace.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	fc, err := foreigncluster.GetForeignClusterByID(ctx, r.kubeconfig.CRClient, plan.ClusterID.Value)
	if kerrors.IsNotFound(err) {
		fc = &discoveryv1alpha1.ForeignCluster{ObjectMeta: metav1.ObjectMeta{Name: plan.ClusterName.Value,
			Labels: map[string]string{discovery.ClusterIDLabel: plan.ClusterID.Value}}}
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.kubeconfig.CRClient, fc, func() error {
		if fc.Spec.PeeringType != discoveryv1alpha1.PeeringTypeUnknown && fc.Spec.PeeringType != discoveryv1alpha1.PeeringTypeOutOfBand {
			return fmt.Errorf("a peering of type %s already exists towards remote cluster %q, cannot be changed to %s",
				fc.Spec.PeeringType, plan.ClusterName.Value, discoveryv1alpha1.PeeringTypeOutOfBand)
		}

		fc.Spec.PeeringType = discoveryv1alpha1.PeeringTypeOutOfBand
		fc.Spec.ClusterIdentity.ClusterID = plan.ClusterID.Value
		if fc.Spec.ClusterIdentity.ClusterName == "" {
			fc.Spec.ClusterIdentity.ClusterName = plan.ClusterName.Value
		}

		fc.Spec.ForeignAuthURL = plan.ClusterAuthURL.Value
		fc.Spec.ForeignProxyURL = ""
		fc.Spec.OutgoingPeeringEnabled = discoveryv1alpha1.PeeringEnabledYes
		if fc.Spec.IncomingPeeringEnabled == "" {
			fc.Spec.IncomingPeeringEnabled = discoveryv1alpha1.PeeringEnabledAuto
		}
		if fc.Spec.InsecureSkipTLSVerify == nil {
			fc.Spec.InsecureSkipTLSVerify = pointer.BoolPtr(true)
		}
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
func (r *peeringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state peeringResourceModel
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
func (r *peeringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *peeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var data peeringResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var foreignCluster discoveryv1alpha1.ForeignCluster
	if err := r.kubeconfig.CRClient.Get(ctx, kubeTypes.NamespacedName{Name: data.ClusterName.Value}, &foreignCluster); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			err.Error(),
		)
		return
	}

	// Do not proceed if the peering is not out-of-band and that mode is set.
	if foreignCluster.Spec.PeeringType != discoveryv1alpha1.PeeringTypeOutOfBand {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			"The peering type towards remote cluster "+data.ClusterName.Value+" is not OOB",
		)
		return
	}

	foreignCluster.Spec.OutgoingPeeringEnabled = discoveryv1alpha1.PeeringEnabledNo
	if err := r.kubeconfig.CRClient.Update(ctx, &foreignCluster); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			err.Error(),
		)
		return
	}

}

// Configure adds the provider configured client to the resource.
func (r *peeringResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.kubeconfig = req.ProviderData.(kubeconfig)
}

// peeringResourceModel maps the resource schema data.
type peeringResourceModel struct {
	ClusterID      types.String `tfsdk:"cluster_id"`
	ClusterName    types.String `tfsdk:"cluster_name"`
	ClusterAuthURL types.String `tfsdk:"cluster_authurl"`
	ClusterToken   types.String `tfsdk:"cluster_token"`
	LiqoNamespace  types.String `tfsdk:"liqo_namespace"`
}
