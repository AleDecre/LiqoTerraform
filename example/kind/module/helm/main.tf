terraform {
  required_providers {
    helm = {
      source                = "hashicorp/helm"
      version               = "2.7.1"
      configuration_aliases = [helm.rome, helm.milan]
    }
  }
}

resource "helm_release" "install_liqo_rome" {
  provider = helm.rome

  name = "liqo"

  repository       = "https://helm.liqo.io/"
  chart            = "liqo"
  namespace        = "liqo"
  create_namespace = true

  set {
    name  = "discovery.config.clusterName"
    value = "liqo"
  }
  set {
    name  = "discovery.config.clusterLabels.liqo\\.io/provider"
    value = "kind"
  }
  set {
    name  = "auth.service.type"
    value = "NodePort"
  }
  set {
    name  = "gateway.service.type"
    value = "NodePort"
  }
  set {
    name  = "networkManager.config.serviceCIDR"
    value = var.clusters_list[0].networking.service_subnet
  }
  set {
    name  = "networkManager.config.podCIDR"
    value = var.clusters_list[0].networking.pod_subnet
  }

}

resource "helm_release" "install_liqo_milan" {
  provider = helm.milan

  name = "liqo"

  repository       = "https://helm.liqo.io/"
  chart            = "liqo"
  namespace        = "liqo"
  create_namespace = true

  set {
    name  = "discovery.config.clusterName"
    value = "liqo"
  }
  set {
    name  = "discovery.config.clusterLabels.liqo\\.io/provider"
    value = var.liqo_provider
  }
  set {
    name  = "auth.service.type"
    value = "NodePort"
  }
  set {
    name  = "gateway.service.type"
    value = "NodePort"
  }
  set {
    name  = "networkManager.config.serviceCIDR"
    value = var.clusters_list[1].networking.service_subnet
  }
  set {
    name  = "networkManager.config.podCIDR"
    value = var.clusters_list[1].networking.pod_subnet
  }

}

