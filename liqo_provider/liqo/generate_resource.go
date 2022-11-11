package liqo

import (
	"context"
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
	"github.com/liqotech/liqo/pkg/auth"
	"github.com/liqotech/liqo/pkg/utils"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
}

// Metadata returns the resource type name.
func (r *generateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generate"
}

// GetSchema defines the schema for the resource.
func (r *generateResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"kubeconfig_path": {
				Type:     types.StringType,
				Required: true,
			},
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
	_ = KubeClient

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, CRClient, "liqo")
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}
	_ = clusterIdentity

	localToken, err := auth.GetToken(ctx, CRClient, "liqo")
	if err != nil {
		plan.ErrorMsg = types.StringValue(err.Error())
	}

	authEP, err := foreigncluster.GetHomeAuthURL(ctx, CRClient, "liqo")
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
}

// generateResourceModel maps the resource schema data.
type generateResourceModel struct {
	KubeconfigPath types.String `tfsdk:"kubeconfig_path"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	ClusterName    types.String `tfsdk:"cluster_name"`
	AuthEP         types.String `tfsdk:"auth_ep"`
	LocalToken     types.String `tfsdk:"local_token"`
	ErrorMsg       types.String `tfsdk:"error_msg"`
}
