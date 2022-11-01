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

resource "kind_cluster" "default" {

  for_each = {
    for cluster in var.clusters :
    cluster.name => cluster if cluster.location == "local"
  }

  name            = each.value.name
  node_image      = "kindest/node:${var.kind_version}"
  kubeconfig_path = "./config/liqo_kubeconf_${each.value.name}"
  wait_for_ready  = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {
      service_subnet = each.value.networking.service_subnet
      pod_subnet     = each.value.networking.pod_subnet
    }
    node {
      role = "control-plane"
    }
  }

  provisioner "local-exec" {
    command = "liqoctl install kind --cluster-name ${self.name}"

    environment = {
      KUBECONFIG = "${self.kubeconfig_path}"
    }
  }

}

resource "kind_cluster" "remote" {

  for_each = {
    for cluster in var.clusters :
    cluster.name => cluster if cluster.location == "remote"
  }

  name            = each.value.name
  node_image      = "kindest/node:${var.kind_version}"
  kubeconfig_path = "./config/liqo_kubeconf_${each.value.name}"
  wait_for_ready  = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {
      service_subnet = each.value.networking.service_subnet
      pod_subnet     = each.value.networking.pod_subnet
    }
    node {
      role = "control-plane"
    }
  }

  provisioner "local-exec" {
    command = "liqoctl install kind --cluster-name ${self.name}"

    environment = {
      KUBECONFIG = "${self.kubeconfig_path}"
    }
  }

}
