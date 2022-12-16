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
    liqo = {
      source = "liqo-provider/liqo/test"
    }
  }
}


provider "helm" {
  alias = "rome"
  kubernetes {
    config_path = kind_cluster.rome.kubeconfig_path
  }

}
provider "helm" {
  alias = "milan"
  kubernetes {
    config_path = kind_cluster.milan.kubeconfig_path
  }

}
provider "liqo" {
  alias = "rome"
  kubernetes = {
    config_path = kind_cluster.rome.kubeconfig_path
  }
}
provider "liqo" {
  alias = "milan"
  kubernetes = {
    config_path = kind_cluster.milan.kubeconfig_path
  }
}


resource "kind_cluster" "rome" {

  name           = "rome"
  node_image     = "kindest/node:v1.23.6"
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {

      service_subnet = "10.90.0.0/12"
      pod_subnet     = "10.200.0.0/16"
    }
    node {
      role = "control-plane"
    }
    node {
      role = "worker"
    }
  }

}
resource "kind_cluster" "milan" {

  name           = "milan"
  node_image     = "kindest/node:v1.23.6"
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {

      service_subnet = "10.90.0.0/12"
      pod_subnet     = "10.200.0.0/16"
    }
    node {
      role = "control-plane"
    }
    node {
      role = "worker"
    }
  }

}


resource "helm_release" "install_liqo_rome" {

  provider = helm.rome

  name = "liqorome"

  repository       = "https://helm.liqo.io/"
  chart            = "liqo"
  namespace        = "liqo"
  create_namespace = true

  set {
    name  = "discovery.config.clusterName"
    value = "rome"
  }
  set {
    name  = "discovery.config.clusterIdOverride"
    value = "cbea6d94-5d1e-4f48-85ad-7eb19e92d7e9"
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
    value = "10.90.0.0/12"
  }
  set {
    name  = "networkManager.config.podCIDR"
    value = "10.200.0.0/16"
  }

}
resource "helm_release" "install_liqo_milan" {

  provider = helm.milan

  name = "liqomilan"

  repository       = "https://helm.liqo.io/"
  chart            = "liqo"
  namespace        = "liqo"
  create_namespace = true

  set {
    name  = "discovery.config.clusterName"
    value = "milan"
  }
  set {
    name  = "discovery.config.clusterIdOverride"
    value = "36148485-d598-4d79-86fe-2559aba68d3c"
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
    value = "10.90.0.0/12"
  }
  set {
    name  = "networkManager.config.podCIDR"
    value = "10.200.0.0/16"
  }

}


resource "liqo_generate" "generate" {

  depends_on = [
    helm_release.install_liqo_milan
  ]

  provider = liqo.milan

}
resource "liqo_peering" "peering" {

  depends_on = [
    helm_release.install_liqo_rome
  ]

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
      KUBECONFIG = kind_cluster.rome.kubeconfig_path
    }
  }

}
resource "liqo_offload" "offload" {

  depends_on = [
    null_resource.create_namespace,
    liqo_peering.peering
  ]

  provider = liqo.rome

  cluster_selector_terms = [
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
