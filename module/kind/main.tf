terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
  }
}

resource "kind_cluster" "default" {

  for_each = {
    for cluster in var.clusters :
    cluster.name => cluster
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
    node {
      role = "worker"
    }
  }

  provisioner "local-exec" {
    command = "liqoctl install kind --cluster-name ${self.name}"

    environment = {
      KUBECONFIG = "${self.kubeconfig_path}"
    }
  }

}

resource "null_resource" "cluster_peering" {

  for_each = {
    for cluster in var.clusters :
    cluster.name => cluster if cluster.remote
  }

  provisioner "local-exec" {

    command = "$(liqoctl generate peer-command --kubeconfig \"$KUBECONFIG_REMOTE\" | tail -n 1)"

    environment = {
      KUBECONFIG       = "${kind_cluster.default["rome"].kubeconfig_path}"
      KUBECONFIG_REMOTE = "${kind_cluster.default[each.value.name].kubeconfig_path}"
    }

  }

}
