terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
  }
}

provider "kind" {
}

module "kind"{
  source = "./module/kind"
  
  clusters = var.clusters
  kind_version = var.kind_version
  
}