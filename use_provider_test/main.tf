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
  id = "./config/liqo_kubeconf_rome"
}

output "edu_order" {
  value = liqo_peering.edu
}
