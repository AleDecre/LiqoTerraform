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
	offloadingv1alpha1 "github.com/liqotech/liqo/apis/offloading/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
	"github.com/mitchellh/go-homedir"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	_ resource.Resource              = &offloadResource{}
	_ resource.ResourceWithConfigure = &offloadResource{}
)

func NewOffloadResource() resource.Resource {
	return &offloadResource{}
}

type offloadResource struct {
	config liqoProviderModel
}

func (o *offloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_offload"
}

func (o *offloadResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Offload a namespace.",
		Attributes: map[string]tfsdk.Attribute{
			"namespace": {
				Type:        types.StringType,
				Required:    true,
				Description: "Offload a namespace.",
			},
			"pod_offloading_strategy": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					attribute_plan_modifier.DefaultValue(types.StringValue("LocalAndRemote")),
				},
				Computed:    true,
				Description: "Namespace to offload.",
			},
			"namespace_mapping_strategy": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					attribute_plan_modifier.DefaultValue(types.StringValue("DefaultName")),
				},
				Computed:    true,
				Description: "Naming strategy used to create the remote namespace.",
			},
			"cluster_selector_terms": {
				Optional: true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"match_expressions": {
						Optional: true,
						Computed: true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"key": {
								Type:        types.StringType,
								Required:    true,
								Description: " The label key that the selector applies to.",
							},
							"operator": {
								Type:        types.StringType,
								Required:    true,
								Description: "Represents a key's relationship to a set of values.",
							},
							"values": {
								Type:        types.ListType{ElemType: types.StringType},
								Optional:    true,
								Description: "An array of string values.",
							},
						}),
						Description: "A list of cluster selector.",
					},
				}),
				Description: "Selectors to restrict the set of remote clusters.",
			},
		},
	}, nil
}

// Creation of Offload Resource to offload a specific namespace,
// additionally there is a possibility to select clusters with match_expressione
// This resource will reproduce the same effect and outputs of "liqoctl offload" command
func (o *offloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan offloadResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if !o.config.KUBERNETES.KUBE_CONFIG_PATH.IsNull() {
		configPaths = []string{o.config.KUBERNETES.KUBE_CONFIG_PATH.ValueString()}
	} else if len(o.config.KUBERNETES.KUBE_CONFIG_PATHS) > 0 {
		for _, configPath := range o.config.KUBERNETES.KUBE_CONFIG_PATHS {
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

		ctxNotOk := o.config.KUBERNETES.KUBE_CTX.IsNull()
		authInfoNotOk := o.config.KUBERNETES.KUBE_CTX_AUTH_INFO.IsNull()
		clusterNotOk := o.config.KUBERNETES.KUBE_CTX_CLUSTER.IsNull()

		if ctxNotOk || authInfoNotOk || clusterNotOk {
			if ctxNotOk {
				overrides.CurrentContext = o.config.KUBERNETES.KUBE_CTX.ValueString()
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoNotOk {
				overrides.Context.AuthInfo = o.config.KUBERNETES.KUBE_CTX_AUTH_INFO.ValueString()
			}
			if clusterNotOk {
				overrides.Context.Cluster = o.config.KUBERNETES.KUBE_CTX_CLUSTER.ValueString()
			}
		}
	}

	if !o.config.KUBERNETES.KUBE_INSECURE.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = !o.config.KUBERNETES.KUBE_INSECURE.ValueBool()
	}
	if !o.config.KUBERNETES.KUBE_CLUSTER_CA_CERT_DATA.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(o.config.KUBERNETES.KUBE_CLUSTER_CA_CERT_DATA.ValueString()).Bytes()
	}
	if !o.config.KUBERNETES.KUBE_CLIENT_CERT_DATA.IsNull() {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(o.config.KUBERNETES.KUBE_CLIENT_CERT_DATA.ValueString()).Bytes()
	}
	if !o.config.KUBERNETES.KUBE_HOST.IsNull() {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(o.config.KUBERNETES.KUBE_HOST.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Resource",
				err.Error(),
			)
			return
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if !o.config.KUBERNETES.KUBE_USER.IsNull() {
		overrides.AuthInfo.Username = o.config.KUBERNETES.KUBE_USER.ValueString()
	}
	if !o.config.KUBERNETES.KUBE_PASSWORD.IsNull() {
		overrides.AuthInfo.Password = o.config.KUBERNETES.KUBE_PASSWORD.ValueString()
	}
	if !o.config.KUBERNETES.KUBE_CLIENT_KEY_DATA.IsNull() {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(o.config.KUBERNETES.KUBE_CLIENT_KEY_DATA.ValueString()).Bytes()
	}
	if !o.config.KUBERNETES.KUBE_TOKEN.IsNull() {
		overrides.AuthInfo.Token = o.config.KUBERNETES.KUBE_TOKEN.ValueString()
	}

	if !o.config.KUBERNETES.KUBE_PROXY_URL.IsNull() {
		overrides.ClusterDefaults.ProxyURL = o.config.KUBERNETES.KUBE_PROXY_URL.ValueString()
	}

	if len(o.config.KUBERNETES.KUBE_EXEC) > 0 {
		exec := &clientcmdapi.ExecConfig{}
		exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		exec.APIVersion = o.config.KUBERNETES.KUBE_EXEC[0].API_VERSION.ValueString()
		exec.Command = o.config.KUBERNETES.KUBE_EXEC[0].COMMAND.ValueString()
		for _, arg := range o.config.KUBERNETES.KUBE_EXEC[0].ARGS {
			exec.Args = append(exec.Args, arg.ValueString())
		}

		for kk, vv := range o.config.KUBERNETES.KUBE_EXEC[0].ENV.Elements() {
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

	var clusterSelector [][]metav1.LabelSelectorRequirement

	for _, selector := range plan.ClusterSelectorTerms {
		s := &metav1.LabelSelector{
			MatchLabels:      map[string]string{},
			MatchExpressions: []metav1.LabelSelectorRequirement{},
		}

		for _, match_expression := range selector.MatchExpressions {

			var values []string

			for _, value := range match_expression.Values {
				values = append(values, value.ValueString())
			}
			req := metav1.LabelSelectorRequirement{
				Key:      match_expression.Key.ValueString(),
				Operator: metav1.LabelSelectorOperator(match_expression.Operator.ValueString()),
				Values:   values,
			}
			s.MatchExpressions = append(s.MatchExpressions, req)
		}

		clusterSelector = append(clusterSelector, s.MatchExpressions)
	}

	terms := []corev1.NodeSelectorTerm{}

	for _, selector := range clusterSelector {
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
		Name: consts.DefaultNamespaceOffloadingName, Namespace: plan.Namespace.ValueString()}}

	_, err = controllerutil.CreateOrUpdate(ctx, CRClient, nsoff, func() error {
		nsoff.Spec.PodOffloadingStrategy = offloadingv1alpha1.PodOffloadingStrategyType(plan.PodOffloadingStrategy.ValueString())
		nsoff.Spec.NamespaceMappingStrategy = offloadingv1alpha1.NamespaceMappingStrategyType(plan.NamespaceMappingStrategy.ValueString())
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

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (o *offloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state offloadResourceModel
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

func (o *offloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unable to Update Resource",
		"Update is not supported/permitted yet.",
	)
}

func (o *offloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var data offloadResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if !o.config.KUBERNETES.KUBE_CONFIG_PATH.IsNull() {
		configPaths = []string{o.config.KUBERNETES.KUBE_CONFIG_PATH.ValueString()}
	} else if len(o.config.KUBERNETES.KUBE_CONFIG_PATHS) > 0 {
		for _, configPath := range o.config.KUBERNETES.KUBE_CONFIG_PATHS {
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

		ctxNotOk := o.config.KUBERNETES.KUBE_CTX.IsNull()
		authInfoNotOk := o.config.KUBERNETES.KUBE_CTX_AUTH_INFO.IsNull()
		clusterNotOk := o.config.KUBERNETES.KUBE_CTX_CLUSTER.IsNull()

		if ctxNotOk || authInfoNotOk || clusterNotOk {
			if ctxNotOk {
				overrides.CurrentContext = o.config.KUBERNETES.KUBE_CTX.ValueString()
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoNotOk {
				overrides.Context.AuthInfo = o.config.KUBERNETES.KUBE_CTX_AUTH_INFO.ValueString()
			}
			if clusterNotOk {
				overrides.Context.Cluster = o.config.KUBERNETES.KUBE_CTX_CLUSTER.ValueString()
			}
		}
	}

	if !o.config.KUBERNETES.KUBE_INSECURE.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = !o.config.KUBERNETES.KUBE_INSECURE.ValueBool()
	}
	if !o.config.KUBERNETES.KUBE_CLUSTER_CA_CERT_DATA.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(o.config.KUBERNETES.KUBE_CLUSTER_CA_CERT_DATA.ValueString()).Bytes()
	}
	if !o.config.KUBERNETES.KUBE_CLIENT_CERT_DATA.IsNull() {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(o.config.KUBERNETES.KUBE_CLIENT_CERT_DATA.ValueString()).Bytes()
	}
	if !o.config.KUBERNETES.KUBE_HOST.IsNull() {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(o.config.KUBERNETES.KUBE_HOST.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Resource",
				err.Error(),
			)
			return
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if !o.config.KUBERNETES.KUBE_USER.IsNull() {
		overrides.AuthInfo.Username = o.config.KUBERNETES.KUBE_USER.ValueString()
	}
	if !o.config.KUBERNETES.KUBE_PASSWORD.IsNull() {
		overrides.AuthInfo.Password = o.config.KUBERNETES.KUBE_PASSWORD.ValueString()
	}
	if !o.config.KUBERNETES.KUBE_CLIENT_KEY_DATA.IsNull() {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(o.config.KUBERNETES.KUBE_CLIENT_KEY_DATA.ValueString()).Bytes()
	}
	if !o.config.KUBERNETES.KUBE_TOKEN.IsNull() {
		overrides.AuthInfo.Token = o.config.KUBERNETES.KUBE_TOKEN.ValueString()
	}

	if !o.config.KUBERNETES.KUBE_PROXY_URL.IsNull() {
		overrides.ClusterDefaults.ProxyURL = o.config.KUBERNETES.KUBE_PROXY_URL.ValueString()
	}

	if len(o.config.KUBERNETES.KUBE_EXEC) > 0 {
		exec := &clientcmdapi.ExecConfig{}
		exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		exec.APIVersion = o.config.KUBERNETES.KUBE_EXEC[0].API_VERSION.ValueString()
		exec.Command = o.config.KUBERNETES.KUBE_EXEC[0].COMMAND.ValueString()
		for _, arg := range o.config.KUBERNETES.KUBE_EXEC[0].ARGS {
			exec.Args = append(exec.Args, arg.ValueString())
		}

		for kk, vv := range o.config.KUBERNETES.KUBE_EXEC[0].ENV.Elements() {
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

	if CRClient == nil {
		return
	}

	nsoff := &offloadingv1alpha1.NamespaceOffloading{ObjectMeta: metav1.ObjectMeta{
		Name: consts.DefaultNamespaceOffloadingName, Namespace: data.Namespace.ValueString()}}
	if err := CRClient.Delete(ctx, nsoff); client.IgnoreNotFound(err) != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Resource",
			err.Error(),
		)
		return
	}

}

// Configure method to obtain kubernetes Clients provided by provider
func (o *offloadResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	o.config = req.ProviderData.(liqoProviderModel)

}

type match_expression struct {
	Key      types.String   `tfsdk:"key"`
	Operator types.String   `tfsdk:"operator"`
	Values   []types.String `tfsdk:"values"`
}

type match_expressions struct {
	MatchExpressions []match_expression `tfsdk:"match_expressions"`
}

type offloadResourceModel struct {
	Namespace                types.String        `tfsdk:"namespace"`
	PodOffloadingStrategy    types.String        `tfsdk:"pod_offloading_strategy"`
	NamespaceMappingStrategy types.String        `tfsdk:"namespace_mapping_strategy"`
	ClusterSelectorTerms     []match_expressions `tfsdk:"cluster_selector_terms"`
}
