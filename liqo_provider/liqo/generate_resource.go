package liqo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"terraform-provider-liqo/liqo/attribute_plan_modifier"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/liqotech/liqo/pkg/auth"
	"github.com/liqotech/liqo/pkg/utils"
	foreigncluster "github.com/liqotech/liqo/pkg/utils/foreignCluster"
	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ resource.Resource              = &generateResource{}
	_ resource.ResourceWithConfigure = &generateResource{}
)

func NewGenerateResource() resource.Resource {
	return &generateResource{}
}

type generateResource struct {
	config     liqoProviderModel
	CRClient   client.Client
	KubeClient *kubernetes.Clientset
}

func (r *generateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generate"
}

func (r *generateResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Generate peering parameters for remote clusters",
		Attributes: map[string]tfsdk.Attribute{
			"cluster_id": {
				Type:        types.StringType,
				Computed:    true,
				Description: "Provider cluster ID.",
			},
			"cluster_name": {
				Type:        types.StringType,
				Computed:    true,
				Description: "Provider cluster name.",
			},
			"auth_ep": {
				Type:        types.StringType,
				Computed:    true,
				Description: "Provider authentication endpoint.",
			},
			"local_token": {
				Type:        types.StringType,
				Computed:    true,
				Description: "Provider authentication token.",
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

// Creation of Generate Resource to obtain necessary pairing parameters used by Peering Resources
// This resource will reproduce the same effect and outputs of "liqoctl generate peer-command" command
func (r *generateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan generateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if !r.config.KUBERNETES.KUBE_CONFIG_PATH.IsNull() {
		configPaths = []string{r.config.KUBERNETES.KUBE_CONFIG_PATH.ValueString()}
	} else if len(r.config.KUBERNETES.KUBE_CONFIG_PATHS) > 0 {
		for _, configPath := range r.config.KUBERNETES.KUBE_CONFIG_PATHS {
			configPaths = append(configPaths, configPath.ValueString())
		}
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Create Resource",
					err.Error(),
				)
				return
			}
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxNotOk := r.config.KUBERNETES.KUBE_CTX.IsNull()
		authInfoNotOk := r.config.KUBERNETES.KUBE_CTX_AUTH_INFO.IsNull()
		clusterNotOk := r.config.KUBERNETES.KUBE_CTX_CLUSTER.IsNull()

		if ctxNotOk || authInfoNotOk || clusterNotOk {
			if ctxNotOk {
				overrides.CurrentContext = r.config.KUBERNETES.KUBE_CTX.ValueString()
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoNotOk {
				overrides.Context.AuthInfo = r.config.KUBERNETES.KUBE_CTX_AUTH_INFO.ValueString()
			}
			if clusterNotOk {
				overrides.Context.Cluster = r.config.KUBERNETES.KUBE_CTX_CLUSTER.ValueString()
			}
		}
	}

	if !r.config.KUBERNETES.KUBE_INSECURE.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = !r.config.KUBERNETES.KUBE_INSECURE.ValueBool()
	}
	if !r.config.KUBERNETES.KUBE_CLUSTER_CA_CERT_DATA.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(r.config.KUBERNETES.KUBE_CLUSTER_CA_CERT_DATA.ValueString()).Bytes()
	}
	if !r.config.KUBERNETES.KUBE_CLIENT_CERT_DATA.IsNull() {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(r.config.KUBERNETES.KUBE_CLIENT_CERT_DATA.ValueString()).Bytes()
	}
	if !r.config.KUBERNETES.KUBE_HOST.IsNull() {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(r.config.KUBERNETES.KUBE_HOST.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Resource",
				err.Error(),
			)
			return
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if !r.config.KUBERNETES.KUBE_USER.IsNull() {
		overrides.AuthInfo.Username = r.config.KUBERNETES.KUBE_USER.ValueString()
	}
	if !r.config.KUBERNETES.KUBE_PASSWORD.IsNull() {
		overrides.AuthInfo.Password = r.config.KUBERNETES.KUBE_PASSWORD.ValueString()
	}
	if !r.config.KUBERNETES.KUBE_CLIENT_KEY_DATA.IsNull() {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(r.config.KUBERNETES.KUBE_CLIENT_KEY_DATA.ValueString()).Bytes()
	}
	if !r.config.KUBERNETES.KUBE_TOKEN.IsNull() {
		overrides.AuthInfo.Token = r.config.KUBERNETES.KUBE_TOKEN.ValueString()
	}

	if !r.config.KUBERNETES.KUBE_PROXY_URL.IsNull() {
		overrides.ClusterDefaults.ProxyURL = r.config.KUBERNETES.KUBE_PROXY_URL.ValueString()
	}

	if len(r.config.KUBERNETES.KUBE_EXEC) > 0 {
		exec := &clientcmdapi.ExecConfig{}
		exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		exec.APIVersion = r.config.KUBERNETES.KUBE_EXEC[0].API_VERSION.ValueString()
		exec.Command = r.config.KUBERNETES.KUBE_EXEC[0].COMMAND.ValueString()
		for _, arg := range r.config.KUBERNETES.KUBE_EXEC[0].ARGS {
			exec.Args = append(exec.Args, arg.ValueString())
		}

		for kk, vv := range r.config.KUBERNETES.KUBE_EXEC[0].ENV.Elements() {
			exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.String()})
		}

		overrides.AuthInfo.Exec = exec
	}

	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if clientCfg == nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			"Unable to Create Resource while creating clientCfg",
		)
		return
	}

	var restCfg *rest.Config

	restCfg, err := clientCfg.ClientConfig()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	var CRClient client.Client

	CRClient, err = client.New(restCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	KubeClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	r.CRClient = CRClient
	r.KubeClient = KubeClient

	clusterIdentity, err := utils.GetClusterIdentityWithControllerClient(ctx, r.CRClient, plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	localToken, err := auth.GetToken(ctx, r.CRClient, plan.LiqoNamespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Resource",
			err.Error(),
		)
		return
	}

	authEP, err := foreigncluster.GetHomeAuthURL(ctx, r.CRClient, plan.LiqoNamespace.ValueString())
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
func (r *generateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.config = req.ProviderData.(liqoProviderModel)

}

type generateResourceModel struct {
	ClusterID     types.String `tfsdk:"cluster_id"`
	ClusterName   types.String `tfsdk:"cluster_name"`
	AuthEP        types.String `tfsdk:"auth_ep"`
	LocalToken    types.String `tfsdk:"local_token"`
	LiqoNamespace types.String `tfsdk:"liqo_namespace"`
}
