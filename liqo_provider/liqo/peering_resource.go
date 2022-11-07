package liqo

import (
	"bytes"
	"context"
	"time"

	"html/template"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/liqotech/liqo/pkg/liqoctl/completion"
	"github.com/liqotech/liqo/pkg/liqoctl/factory"
	"github.com/liqotech/liqo/pkg/liqoctl/output"
	"github.com/liqotech/liqo/pkg/liqoctl/peer"
	"github.com/liqotech/liqo/pkg/liqoctl/peeroob"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/util"
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

var liqoctl string

const liqoctlLongHelp = `{{ .Executable}} is a CLI tool to install and manage Liqo.
Liqo is a platform to enable dynamic and decentralized resource sharing across
Kubernetes clusters, either on-prem or managed. Liqo allows to run pods on a
remote cluster seamlessly and without any modification of Kubernetes and the
applications. With Liqo it is possible to extend the control and data plane of a
Kubernetes cluster across the cluster's boundaries, making multi-cluster native
and transparent: collapse an entire remote cluster to a local virtual node,
enabling workloads offloading, resource management and cross-cluster communication
compliant with the standard Kubernetes approach.
`

const liqoctlPeerLongHelp = `Enable a peering towards a remote cluster.
In Liqo, a *peering* is a unidirectional resource and service consumption
relationship between two Kubernetes clusters, with one cluster (i.e., the
consumer) granted the capability to offload tasks in a remote cluster (i.e., the
provider), but not vice versa. Bidirectional peerings can be achieved through
their combination. The same cluster can play the role of provider and consumer
in multiple peerings.
Liqo supports two peering approaches, respectively referred to as out-of-band
control-plane and in-band control-plane. In the *out-of-band* control plane
peering, the Liqo control plane traffic flows outside the VPN tunnel used for
cross-cluster pod-to-pod communication. With the *in-band* approach, on the other
hand, all control plane traffic flows inside the VPN tunnel. The latter approach
relaxes the connectivity requirements, although it requires access to both
clusters (i.e., kubeconfigs) to start the peering process and setup the VPN tunnel.
This command enables a peering towards an already known remote cluster, without the
need of specifying all authentication parameters. It adopts the same approach already
used while peering for the first time with the given remote cluster.
Warning: the establishment of a peering with a remote cluster leveraging a different
version of Liqo, net of patch releases, is currently *not supported*, and could
lead to unexpected results.
Examples:
  $ {{ .Executable }} peer eternal-donkey
or
  $ {{ .Executable }} peer nearby-malamute --namespace liqo-system
`

const liqoctlPeerOOBLongHelp = `Enable an out-of-band peering towards a remote cluster.
The out-of-band control plane peering is the standard peering approach, with the
Liqo control-plane traffic flowing outside the VPN tunnel interconnecting the
two clusters. The VPN tunnel is dynamically started in a later stage of the
peering process, and it is leveraged only for cross-cluster pods traffic.
This approach supports clusters under the control of different administrative
domains (i.e., only local cluster access is required), and it is characterized
by higher dynamism and resilience in case of reconfigurations. Yet, it requires
three different endpoints to be reachable from the pods running in the remote
cluster (i.e., the Liqo authentication service, the Liqo VPN endpoint and the
Kubernetes API server).
Examples:
  $ {{ .Executable }} peer out-of-band eternal-donkey --auth-url <auth-url> \
      --cluster-id <cluster-id> --auth-token <auth-token>
or
  $ {{ .Executable }} peer out-of-band nearby-malamute --auth-url <auth-url> \
      --cluster-id <cluster-id> --auth-token <auth-token> --namespace liqo-system
The command above can be generated executing the following from the target cluster:
  $ {{ .Executable }} generate peer-command
`

func WithTemplate(str string) string {
	tmpl := template.Must(template.New("liqoctl").Parse(str))
	var buf bytes.Buffer
	util.CheckErr(tmpl.Execute(&buf, struct{ Executable string }{liqoctl}))
	return buf.String()
}

// singleClusterPersistentPreRun initializes the local factory.
func singleClusterPersistentPreRun(_ *cobra.Command, f *factory.Factory, opts ...factory.Options) {
	// Populate the factory fields based on the configured parameters.
	f.Printer.CheckErr(f.Initialize(opts...))
}

func newPeerOutOfBandCommand(ctx context.Context, peerOptions *peer.Options) *cobra.Command {
	options := &peeroob.Options{Options: peerOptions}
	cmd := &cobra.Command{
		Use:     "out-of-band cluster-name",
		Aliases: []string{"oob"},
		Short:   "Enable an out-of-band peering towards a remote cluster",
		Long:    WithTemplate(liqoctlPeerOOBLongHelp),
		Args:    cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			options.ClusterName = "milan"
			output.ExitOnErr(options.Run(ctx))
		},
	}

	cmd.Flags().StringVar(&options.ClusterAuthURL, peeroob.AuthURLFlagName, "https://172.19.0.3:30615",
		"The authentication URL of the target remote cluster")
	cmd.Flags().StringVar(&options.ClusterToken, peeroob.ClusterTokenFlagName, "a0a86c0a104ed9cb7e707a06062ee4881f9fcba4522724cfd6029340e4f62abc3b66806fcc9dc45eb90b6d58f43ea9323b6786d198f07239482c293ff6bd2ee7",
		"The authentication token of the target remote cluster")
	cmd.Flags().StringVar(&options.ClusterID, peeroob.ClusterIDFlagName, "cc389391-70e7-4043-9b8f-ffa3d8c1e954",
		"The Cluster ID identifying the target remote cluster")

	f := peerOptions.Factory
	f.AddLiqoNamespaceFlag(cmd.Flags())
	f.Printer.CheckErr(cmd.RegisterFlagCompletionFunc(factory.FlagNamespace, completion.Namespaces(ctx, f, completion.NoLimit)))

	f.Printer.CheckErr(cmd.MarkFlagRequired(peeroob.ClusterIDFlagName))
	f.Printer.CheckErr(cmd.MarkFlagRequired(peeroob.ClusterTokenFlagName))
	f.Printer.CheckErr(cmd.MarkFlagRequired(peeroob.AuthURLFlagName))

	return cmd
}

func newPeerCommand(ctx context.Context, f *factory.Factory) *cobra.Command {
	options := &peer.Options{Factory: f}
	cmd := &cobra.Command{
		Use:               "peer",
		Short:             "Enable a peering towards a remote cluster",
		Long:              WithTemplate(liqoctlPeerLongHelp),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.ForeignClusters(ctx, f, 1),

		Run: func(cmd *cobra.Command, args []string) {
			options.ClusterName = "milan"
			output.ExitOnErr(options.Run(ctx))
		},
	}

	cmd.PersistentFlags().DurationVar(&options.Timeout, "timeout", 120*time.Second, "Timeout for peering completion")

	cmd.AddCommand(newPeerOutOfBandCommand(ctx, options))

	return cmd
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

	f := factory.NewForLocal()

	cmd := &cobra.Command{
		Use:          liqoctl,
		Short:        "A CLI tool to install and manage Liqo",
		Long:         WithTemplate(liqoctlLongHelp),
		Args:         cobra.NoArgs,
		SilenceUsage: true,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd != nil && cmd.Name() != cobra.ShellCompRequestCmd {
				singleClusterPersistentPreRun(cmd, f)
			}
		},
	}

	cmd.PersistentFlags().String("kubeconfig", plan.ID.Value, "kubeconfig-path")

	cmd.AddCommand(newPeerCommand(ctx, f))

	str, err := cmd.PersistentFlags().GetString("kubeconfig")

	if err != nil {
		plan.LastUpdated = types.StringValue("Error")
		return
	}

	plan.LastUpdated = types.StringValue(str)

	if err := cmd.ExecuteContext(ctx); err != nil {
		plan.LastUpdated = types.StringValue("Error")
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
