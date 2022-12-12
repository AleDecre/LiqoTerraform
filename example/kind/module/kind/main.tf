terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
  }
}

resource "kind_cluster" "default" {

  name            = var.cluster.name
  node_image      = "kindest/node:${var.kind_version}"
  //kubeconfig_path = "./config/liqo_kubeconf_${var.cluster.name}"
  wait_for_ready  = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    networking {
      service_subnet = var.cluster.networking.service_subnet
      pod_subnet     = var.cluster.networking.pod_subnet
    }
    node {
      role = "control-plane"
    }
    node {
      role = "worker"
    }
  }

  provisioner "local-exec" {
    command = "./scripts/load_images.sh ${self.name}; liqoctl install kind --cluster-name ${var.cluster.name} --set telemetry.enable=false"
    environment = {
      KUBECONFIG = "${kind_cluster.default.kubeconfig_path}"
    }
  }

}
