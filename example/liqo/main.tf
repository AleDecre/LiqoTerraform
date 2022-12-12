terraform {
  required_providers {
    liqo = {
      source = "liqo-provider/liqo/test"
    }
  }
}

data "terraform_remote_state" "kind" {
  backend = "local"

  config = {
    path = "/home/adc/cloudProject/liqo-terraform/example/kind/terraform.tfstate"
  }
}



provider "liqo" {
  alias = "rome"
  kubernetes = {
    config_path = data.terraform_remote_state.kind.outputs.kubeconfig_path_rome
  }
}

provider "liqo" {
  alias = "milan"
  kubernetes = {
    config_path = data.terraform_remote_state.kind.outputs.kubeconfig_path_milan
  }
}

resource "liqo_generate" "generate" {

  provider = liqo.milan

}

resource "liqo_peering" "peering" {

  provider = liqo.rome

  cluster_id      = liqo_generate.generate.cluster_id
  cluster_name    = liqo_generate.generate.cluster_name
  cluster_authurl = liqo_generate.generate.auth_ep
  cluster_token   = liqo_generate.generate.local_token

}


resource "time_sleep" "wait_10_seconds" {
  depends_on = [liqo_peering.peering]

  create_duration = "10s"
}


resource "null_resource" "create_namespace" {

  depends_on = [
    time_sleep.wait_10_seconds
  ]

  provisioner "local-exec" {
    command = "kubectl create namespace liqo-demo && kubectl label nodes liqo-milan disktype=ssd"
    environment = {
      KUBECONFIG = data.terraform_remote_state.kind.outputs.kubeconfig_path_rome
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
      match_expressions = [
        {
          key      = "disktype"
          operator = "In"
          values   = ["ssd", "hdd"]
        },
      ]
    }
  ]

  namespace = "liqo-demo"
}
