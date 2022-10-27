terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.14"
    }
  }
}

provider "kind" {
}

resource "kind_cluster" "default" {
  name           = "rome"
  node_image     = var.kind_version
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {
      service_subnet = var.networking_local.service_subnet
      pod_subnet     = var.networking_local.pod_subnet
    }
    node {
      role = "control-plane"
    }
    node {
      role = "worker"
    }
  }

  provisioner "local-exec" {
    command = " liqoctl install kind --cluster-name rome"

    environment = {
      KUBECONFIG = "${kind_cluster.default.kubeconfig_path}"
    }
  }

}

resource "kind_cluster" "milan" {
  name           = "milan"
  node_image     = var.kind_version
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {
      service_subnet = var.networking_remote.service_subnet
      pod_subnet     = var.networking_remote.pod_subnet
    }
    node {
      role = "control-plane"
    }
    node {
      role = "worker"
    }
  }

  provisioner "local-exec" {
    command = "liqoctl install kind --cluster-name milan --kubeconfig \"$KUBECONFIG_MILAN\""

    environment = {
      KUBECONFIG_MILAN = "${kind_cluster.milan.kubeconfig_path}"
    }
  }

}

resource "null_resource" "cluster_peering" {

  count = var.peering

  depends_on = [
    kind_cluster.default,
    kind_cluster.milan
  ]

  provisioner "local-exec" {

    command = "$(liqoctl generate peer-command --kubeconfig \"$KUBECONFIG_MILAN\" | tail -n 1)"

    environment = {
      KUBECONFIG       = "${kind_cluster.default.kubeconfig_path}"
      KUBECONFIG_MILAN = "${kind_cluster.milan.kubeconfig_path}"
    }

  }

}
