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
	"github.com/liqotech/liqo/pkg/discovery"
	"github.com/liqotech/liqo/pkg/utils"
	authenticationtokenutils "github.com/liqotech/liqo/pkg/utils/authenticationtoken"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			"id": {
				Type:     types.StringType,
				Required: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					resource.UseStateForUnknown(),
				},
			},
			"last_updated": {
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

	byte, err := ioutil.ReadFile(plan.ID.Value)
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}

	plan.LastUpdated = types.StringValue(string(byte))

	var clientCfg clientcmd.ClientConfig

	clientCfg, err = clientcmd.NewClientConfigFromBytes(byte)
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}

	var restCfg *rest.Config

	restCfg, err = clientCfg.ClientConfig()
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}

	var CRClient client.Client

	CRClient, err = client.New(restCfg, client.Options{})
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}
	_ = CRClient

	KubeClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, CRClient, "liqo")
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}
	_ = clusterIdentity

	if clusterIdentity.ClusterID == "<ClusterID>" {
		plan.LastUpdated = types.StringValue("Same ClusterID")
	}

	err = authenticationtokenutils.StoreInSecret(ctx, KubeClient, "<ClusterID>", "<ClusterToken>", "liqo")
	if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}

	fc, err := foreigncluster.GetForeignClusterByID(ctx, CRClient, "<ClusterID>")
	if kerrors.IsNotFound(err) {
		fc = &discoveryv1alpha1.ForeignCluster{ObjectMeta: metav1.ObjectMeta{Name: "<ClusterName>",
			Labels: map[string]string{discovery.ClusterIDLabel: "<ClusterID>"}}}
	} else if err != nil {
		plan.LastUpdated = types.StringValue(err.Error())
	}
	_ = fc

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

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"`
}
