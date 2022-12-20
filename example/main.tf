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
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.16.1"
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
provider "kubernetes" {
  config_path = kind_cluster.rome.kubeconfig_path
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
resource "kubernetes_namespace" "namespace" {

  depends_on = [
    kind_cluster.rome
  ]
  metadata {
    name = "liqo-demo"
  }

}
resource "liqo_offload" "offload" {

  depends_on = [
    helm_release.install_liqo_rome,
    kubernetes_namespace.namespace
  ]

  provider = liqo.rome

  namespace = "liqo-demo"

}


resource "kubernetes_pod" "pod_nginx_local" {

  depends_on = [
    liqo_peering.peering,
    liqo_offload.offload
  ]
  metadata {
    labels = {
      app = "liqo-demo"
    }
    name      = "nginx-local"
    namespace = "liqo-demo"
  }

  spec {
    affinity {
      node_affinity {
        required_during_scheduling_ignored_during_execution {
          node_selector_term {
            match_expressions {
              key      = "liqo.io/type"
              operator = "NotIn"
              values = [
                "virtual-node",
              ]
            }
          }
        }
      }
    }
    container {
      name              = "nginx"
      image             = "nginxdemos/hello"
      image_pull_policy = "IfNotPresent"
      port {
        container_port = 80
        name           = "web"
      }
    }
  }
}
resource "kubernetes_pod" "pod_nginx_remote" {

  depends_on = [
    liqo_peering.peering,
    liqo_offload.offload
  ]

  metadata {
    labels = {
      app = "liqo-demo"
    }
    name      = "nginx-remote"
    namespace = "liqo-demo"
  }

  spec {
    affinity {
      node_affinity {
        required_during_scheduling_ignored_during_execution {
          node_selector_term {
            match_expressions {
              key      = "liqo.io/type"
              operator = "In"
              values = [
                "virtual-node",
              ]
            }
          }
        }
      }
    }
    container {
      name              = "nginx"
      image             = "nginxdemos/hello"
      image_pull_policy = "IfNotPresent"
      port {
        container_port = 80
        name           = "web"
      }
    }
  }
}
resource "kubernetes_service" "service_liqo_demo" {

  depends_on = [
    liqo_peering.peering,
    liqo_offload.offload
  ]

  metadata {
    name      = "liqo-demo"
    namespace = "liqo-demo"
  }

  spec {
    port {
      name       = "web"
      port       = 80
      protocol   = "TCP"
      target_port = "web"
    }
    selector = {
      app = "liqo-demo"
    }
    type = "ClusterIP"

  }
}