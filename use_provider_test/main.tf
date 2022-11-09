terraform {
  required_providers {
    liqo = {
      source = "liqo-provider/liqo/test"
    }
  }
}

provider "liqo" {
}

resource "liqo_peering" "edu" {
  kubeconfig_path = "../infrastructure/config/liqo_kubeconf_rome"
  cluster_id      = "<ClusterID>"
  cluster_authurl = "<ClusterAuthURL>"
  cluster_token   = "<ClusterToken>"
}

/*
  output "edu_order" {
    value = liqo_peering.edu
  }
*/
