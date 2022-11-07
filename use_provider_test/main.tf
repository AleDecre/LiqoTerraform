terraform {
  required_providers {
    liqo = {
      source = "liqo-provider/liqo/test"
    }
  }
  required_version = ">= 1.1.0"
}

provider "liqo" {
}

resource "liqo_peering" "edu" {
  id = 10
}

output "edu_order" {
  value = liqo_peering.edu
}
