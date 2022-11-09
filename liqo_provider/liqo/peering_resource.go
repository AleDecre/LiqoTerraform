package liqo

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
	netv1alpha1 "github.com/liqotech/liqo/apis/net/v1alpha1"
	offloadingv1alpha1 "github.com/liqotech/liqo/apis/offloading/v1alpha1"
	sharingv1alpha1 "github.com/liqotech/liqo/apis/sharing/v1alpha1"
	"github.com/liqotech/liqo/pkg/discovery"
	"github.com/liqotech/liqo/pkg/utils"
	authenticationtokenutils "github.com/liqotech/liqo/pkg/utils/authenticationtoken"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
}

// Metadata returns the resource type name.
func (r *peeringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peering"
}

// GetSchema defines the schema for the resource.
func (r *peeringResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"kubeconfig_path": {
				Type:     types.StringType,
				Required: true,
			},
			"cluster_id": {
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
			"error_msg": {
				Type:     types.StringType,
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

	utilruntime.Must(discoveryv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(netv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(offloadingv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(sharingv1alpha1.AddToScheme(scheme.Scheme))

	byte, err := ioutil.ReadFile(plan.KubeconfigPath.Value)
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	var clientCfg clientcmd.ClientConfig

	clientCfg, err = clientcmd.NewClientConfigFromBytes(byte)
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	var restCfg *rest.Config

	restCfg, err = clientCfg.ClientConfig()
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	var CRClient client.Client

	CRClient, err = client.New(restCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}
	_ = CRClient

	KubeClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, CRClient, "liqo")
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}
	_ = clusterIdentity

	if clusterIdentity.ClusterID == plan.ClusterID.Value {
		plan.ErrorMsg = types.StringValue("Same ClusterID")
	}

	err = authenticationtokenutils.StoreInSecret(ctx, KubeClient, plan.ClusterID.Value, plan.ClusterToken.Value, "liqo")
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	fc, err := foreigncluster.GetForeignClusterByID(ctx, CRClient, plan.ClusterID.Value)
	if kerrors.IsNotFound(err) {
		fc = &discoveryv1alpha1.ForeignCluster{ObjectMeta: metav1.ObjectMeta{Name: "milan",
			Labels: map[string]string{discovery.ClusterIDLabel: plan.ClusterID.Value}}}
	} else if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	_, err = controllerutil.CreateOrUpdate(ctx, CRClient, fc, func() error {
		if fc.Spec.PeeringType != discoveryv1alpha1.PeeringTypeUnknown && fc.Spec.PeeringType != discoveryv1alpha1.PeeringTypeOutOfBand {
			return fmt.Errorf("a peering of type %s already exists towards remote cluster %q, cannot be changed to %s",
				fc.Spec.PeeringType, "milan", discoveryv1alpha1.PeeringTypeOutOfBand)
		}

		fc.Spec.PeeringType = discoveryv1alpha1.PeeringTypeOutOfBand
		fc.Spec.ClusterIdentity.ClusterID = plan.ClusterID.Value
		if fc.Spec.ClusterIdentity.ClusterName == "" {
			fc.Spec.ClusterIdentity.ClusterName = "milan"
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
		plan.ErrorMsg = types.StringValue(err.Error())
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
	// Retrieve values from plan
	var plan peeringResourceModel
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
func (r *peeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// Configure adds the provider configured client to the resource.
func (r *peeringResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
}

// peeringResourceModel maps the resource schema data.
type peeringResourceModel struct {
	KubeconfigPath types.String `tfsdk:"kubeconfig_path"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	ClusterAuthURL types.String `tfsdk:"cluster_authurl"`
	ClusterToken   types.String `tfsdk:"cluster_token"`
	ErrorMsg       types.String `tfsdk:"error_msg"`
}
