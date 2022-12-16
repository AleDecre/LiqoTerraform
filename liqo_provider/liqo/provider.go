package liqo

import (
	"context"
	"terraform-provider-test/liqo/attribute_plan_modifier"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
	netv1alpha1 "github.com/liqotech/liqo/apis/net/v1alpha1"
	offloadingv1alpha1 "github.com/liqotech/liqo/apis/offloading/v1alpha1"
	sharingv1alpha1 "github.com/liqotech/liqo/apis/sharing/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubectl/pkg/scheme"
)

func init() {
	utilruntime.Must(discoveryv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(netv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(offloadingv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(sharingv1alpha1.AddToScheme(scheme.Scheme))
}

var (
	_ provider.Provider = &liqoProvider{}
)

func New() provider.Provider {
	return &liqoProvider{}
}

type liqoProvider struct {
}

func (p *liqoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "liqo"
}

func (p *liqoProvider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Interact with Liqo.",
		Attributes: map[string]tfsdk.Attribute{
			"kubernetes": {
				Optional: true,
				Computed: true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"host": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "The hostname (in form of URI) of Kubernetes master.",
					},
					"username": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
					},
					"password": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
					},
					"insecure": {
						Type:     types.BoolType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.BoolValue(false)),
						},
						Description: "Whether server should be accessed without verifying the TLS certificate.",
					},
					"client_certificate": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "PEM-encoded client certificate for TLS authentication.",
					},
					"client_key": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "PEM-encoded client certificate key for TLS authentication.",
					},
					"cluster_ca_certificate": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "PEM-encoded root certificates bundle for TLS authentication.",
					},
					"config_paths": {
						Type:     types.ListType{ElemType: types.StringType},
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.ListNull(types.StringType)),
						},
					},
					"config_path": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
					},
					"config_context": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
					},
					"config_context_auth_info": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "",
					},
					"config_context_cluster": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "",
					},
					"token": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "Token to authenticate an service account",
					},
					"proxy_url": {
						Type:     types.StringType,
						Optional: true,
						PlanModifiers: []tfsdk.AttributePlanModifier{
							attribute_plan_modifier.DefaultValue(types.StringValue("")),
						},
						Description: "URL to the proxy to be used for all API requests",
					},
					"exec": {
						Optional: true,
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"api_version": {
								Type:     types.StringType,
								Required: true,
								PlanModifiers: []tfsdk.AttributePlanModifier{
									attribute_plan_modifier.DefaultValue(types.StringValue("")),
								},
								Validators: []tfsdk.AttributeValidator{
									stringvalidator.NoneOf("client.authentication.k8s.io/v1alpha1"),
								},
							},
							"command": {
								Type:     types.StringType,
								Required: true,
								PlanModifiers: []tfsdk.AttributePlanModifier{
									attribute_plan_modifier.DefaultValue(types.StringValue("")),
								},
							},
							"env": {
								Type:     types.MapType{ElemType: types.StringType},
								Optional: true,
								PlanModifiers: []tfsdk.AttributePlanModifier{
									attribute_plan_modifier.DefaultValue(types.MapNull(types.StringType)),
								},
							},
							"args": {
								Type:     types.ListType{ElemType: types.StringType},
								Optional: true,
								PlanModifiers: []tfsdk.AttributePlanModifier{
									attribute_plan_modifier.DefaultValue(types.ListNull(types.StringType)),
								},
							},
						}),
					},
				}),
			},
		},
	}, nil
}

// Configure method to create the two kubernetes Clients using parameters passed in the provider instantiation in Terraform main
// After the creation both Clients will be available in resources and data sources
func (p *liqoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config liqoProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.ResourceData = config
}

func (p *liqoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *liqoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPeeringResource, NewGenerateResource, NewOffloadResource,
	}
}

type exec struct {
	API_VERSION types.String   `tfsdk:"api_version"`
	COMMAND     types.String   `tfsdk:"command"`
	ENV         types.Map      `tfsdk:"env"`
	ARGS        []types.String `tfsdk:"args"`
}

type kube_conf struct {
	KUBE_HOST                 types.String   `tfsdk:"host"`
	KUBE_USER                 types.String   `tfsdk:"username"`
	KUBE_PASSWORD             types.String   `tfsdk:"password"`
	KUBE_INSECURE             types.Bool     `tfsdk:"insecure"`
	KUBE_CLIENT_CERT_DATA     types.String   `tfsdk:"client_certificate"`
	KUBE_CLIENT_KEY_DATA      types.String   `tfsdk:"client_key"`
	KUBE_CLUSTER_CA_CERT_DATA types.String   `tfsdk:"cluster_ca_certificate"`
	KUBE_CONFIG_PATH          types.String   `tfsdk:"config_path"`
	KUBE_CONFIG_PATHS         []types.String `tfsdk:"config_paths"`
	KUBE_CTX                  types.String   `tfsdk:"config_context"`
	KUBE_CTX_AUTH_INFO        types.String   `tfsdk:"config_context_auth_info"`
	KUBE_CTX_CLUSTER          types.String   `tfsdk:"config_context_cluster"`
	KUBE_TOKEN                types.String   `tfsdk:"token"`
	KUBE_PROXY_URL            types.String   `tfsdk:"proxy_url"`
	KUBE_EXEC                 []exec         `tfsdk:"exec"`
}

type liqoProviderModel struct {
	KUBERNETES *kube_conf `tfsdk:"kubernetes"`
}
