terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
    liqo = {
      source = "liqo-provider/liqo/test"
    }
  }
}

provider "liqo" {
  alias = "rome"
  kubernetes = {
    config_path = module.kind["rome"].kubeconfig_path
  }
}

provider "liqo" {
  alias = "milan"
  kubernetes = {
    config_path = module.kind["milan"].kubeconfig_path
  }
}

module "kind" {

  for_each = {
    for cluster in var.clusters.clusters_list :
    cluster.name => cluster
  }

  source = "./module/kind"

  cluster      = each.value
  kind_version = var.kind_version

}

resource "liqo_generate" "generate" {
  /*
    for_each = {
      for index, cluster in var.clusters.clusters_list :
      cluster.name => cluster if index != 0 && var.clusters.peering
    }
  */

  provider = liqo.milan

}

resource "liqo_peering" "peering" {

  /*
    for_each = {
      for index, cluster in var.clusters.clusters_list :
      cluster.name => cluster if index != 0 && var.clusters.peering
    }
  */

  provider = liqo.rome

  cluster_id      = liqo_generate.generate.cluster_id
  cluster_name    = liqo_generate.generate.cluster_name
  cluster_authurl = liqo_generate.generate.auth_ep
  cluster_token   = liqo_generate.generate.local_token

}

resource "null_resource" "create_namespace" {

  provisioner "local-exec" {
    command = "kubectl create namespace liqo-demo"
    environment = {
      KUBECONFIG = module.kind["rome"].kubeconfig_path
    }
  }

}

resource "liqo_offload" "offload" {
  depends_on = [
    null_resource.create_namespace,
    liqo_peering.peering
  ]

  provider = liqo.rome

  node_selector_terms = [
    {
      node_selector_term = [
        {
          key      = "disktype"
          operator = "In"
          values   = "ssd"
        },
        {
          key      = "cputype"
          operator = "In"
          values   = "intel"
        }
      ]
      node_selector_term = [
        {
          key      = "ramtype"
          operator = "In"
          values   = "ddr4"
        },
      ]
    },
    {
      node_selector_term = [
        {
          key      = "disktype"
          operator = "In"
          values   = "hdd"
        },
        {
          key      = "cputype"
          operator = "In"
          values   = "arm"
        }
      ]
    }
  ]

  /*


      {
        key      = "disktype"
        operator = "In"
        values   = "ssd"
      },
      {
        key      = "disktype"
        operator = "In"
        values   = "ssd"
      }





    "node_selector": {
				Type:     types.ListType{},
				Optional: true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"node_selector_term": {
						Optional: true,
						Computed: true,
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"match_expression": {
								Type:     types.ListType{},
								Optional: true,
								Computed: true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"key": {
										Type:     types.StringType,
										Required: true,
									},
									"operator": {
										Type:     types.StringType,
										Required: true,
									},
									"values": {
										Type:     types.ListType{ElemType: types.StringType},
										Optional: true,
										PlanModifiers: []tfsdk.AttributePlanModifier{
											attribute_plan_modifier.DefaultValue(types.StringValue("")),
										},
										Computed: true,
									},
								}),
							},
						}),
					},
				}),
			},
  
  */

  namespace = "liqo-demo"
}
