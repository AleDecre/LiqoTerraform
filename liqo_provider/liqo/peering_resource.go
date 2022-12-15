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

var (
	_ resource.Resource              = &peeringResource{}
	_ resource.ResourceWithConfigure = &peeringResource{}
)

func NewPeeringResource() resource.Resource {
	return &peeringResource{}
}

type peeringResource struct {
	kubeconfig kubeconfig
}

func (p *peeringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peering"
}

func (p *peeringResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Execute peering.",
		Attributes: map[string]tfsdk.Attribute{
			"cluster_id": {
				Type:        types.StringType,
				Required:    true,
				Description: "Provider cluster ID used for peering.",
			},
			"cluster_name": {
				Type:        types.StringType,
				Required:    true,
				Description: "Provider cluster name used for peering.",
			},
			"cluster_authurl": {
				Type:        types.StringType,
				Required:    true,
				Description: "Provider authentication url used for peering.",
			},
			"cluster_token": {
				Type:        types.StringType,
				Required:    true,
				Description: "Provider authentication token used for peering.",
			},
			"liqo_namespace": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					attribute_plan_modifier.DefaultValue(types.StringValue("liqo")),
				},
				Computed:    true,
				Description: "Namespace where is Liqo installed in provider cluster.",
			},
		},
	}, nil
}

// Creation of Peering Resource to execute peering between two clusters using auth parameters provided by Generate Resource
// This resource will reproduce the same effect and outputs of "liqoctl peer out-of-band" command
func (p *peeringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan peeringResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, p.kubeconfig.CRClient, plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	if clusterIdentity.ClusterID == plan.ClusterID.ValueString() {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			"The Cluster ID of the remote cluster is the same of that of the local cluster",
		)
		return
	}

	err = authenticationtokenutils.StoreInSecret(ctx, p.kubeconfig.KubeClient, plan.ClusterID.ValueString(), plan.ClusterToken.ValueString(), plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	fc, err := foreigncluster.GetForeignClusterByID(ctx, p.kubeconfig.CRClient, plan.ClusterID.ValueString())
	if kerrors.IsNotFound(err) {
		fc = &discoveryv1alpha1.ForeignCluster{ObjectMeta: metav1.ObjectMeta{Name: plan.ClusterName.ValueString(),
			Labels: map[string]string{discovery.ClusterIDLabel: plan.ClusterID.ValueString()}}}
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	_, err = controllerutil.CreateOrUpdate(ctx, p.kubeconfig.CRClient, fc, func() error {
		if fc.Spec.PeeringType != discoveryv1alpha1.PeeringTypeUnknown && fc.Spec.PeeringType != discoveryv1alpha1.PeeringTypeOutOfBand {
			return fmt.Errorf("a peering of type %s already exists towards remote cluster %q, cannot be changed to %s",
				fc.Spec.PeeringType, plan.ClusterName.ValueString(), discoveryv1alpha1.PeeringTypeOutOfBand)
		}

		fc.Spec.PeeringType = discoveryv1alpha1.PeeringTypeOutOfBand
		fc.Spec.ClusterIdentity.ClusterID = plan.ClusterID.ValueString()
		if fc.Spec.ClusterIdentity.ClusterName == "" {
			fc.Spec.ClusterIdentity.ClusterName = plan.ClusterName.ValueString()
		}

		fc.Spec.ForeignAuthURL = plan.ClusterAuthURL.ValueString()
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

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (p *peeringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state peeringResourceModel
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

func (p *peeringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unable to Update Resource",
		"Update is not supported/permitted yet.",
	)
}

func (p *peeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var data peeringResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var foreignCluster discoveryv1alpha1.ForeignCluster
	if err := p.kubeconfig.CRClient.Get(ctx, kubeTypes.NamespacedName{Name: data.ClusterName.ValueString()}, &foreignCluster); err != nil {
		return
	}

	if foreignCluster.Spec.PeeringType != discoveryv1alpha1.PeeringTypeOutOfBand {
		return
	}

	foreignCluster.Spec.OutgoingPeeringEnabled = discoveryv1alpha1.PeeringEnabledNo
	if err := p.kubeconfig.CRClient.Update(ctx, &foreignCluster); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			err.Error(),
		)
		return
	}

}

// Configure method to obtain kubernetes Clients provided by provider
func (p *peeringResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	p.kubeconfig = req.ProviderData.(kubeconfig)
}

type peeringResourceModel struct {
	ClusterID      types.String `tfsdk:"cluster_id"`
	ClusterName    types.String `tfsdk:"cluster_name"`
	ClusterAuthURL types.String `tfsdk:"cluster_authurl"`
	ClusterToken   types.String `tfsdk:"cluster_token"`
	LiqoNamespace  types.String `tfsdk:"liqo_namespace"`
}
