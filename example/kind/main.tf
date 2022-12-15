terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.7.1"
    }
  }
}

provider "helm" {
  alias = "rome"
  kubernetes {
    config_path = module.kind["rome"].kubeconfig_path
  }

}
provider "helm" {
  alias = "milan"
  kubernetes {
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
module "helm" {

  providers = {
    helm.rome  = helm.rome
    helm.milan = helm.milan
  }

  source = "./module/helm"

  liqo_provider = "kind"
  clusters_list = var.clusters.clusters_list

}
