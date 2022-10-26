terraform {
  required_providers {
    kind = {
      source  = "unicell/kind"
      version = "0.0.2-u2"
    }
  }
}

provider "kind" {
}

resource "kind_cluster" "default" {
  name        = "rome"
  node_image  = "kindest/node:v1.23.6"
  kind_config = <<KIONF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  serviceSubnet: "10.90.0.0/12"
  podSubnet: "10.200.0.0/16"
nodes:
  - role: control-plane
  - role: worker
KIONF

  provisioner "local-exec" {
    command = " liqoctl install kind --cluster-name rome --kubeconfig './rome-config'"
  }
}

resource "kind_cluster" "milan" {
  name        = "milan"
  node_image  = "kindest/node:v1.23.6"
  kind_config = <<KIONF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  serviceSubnet: "10.90.0.0/12"
  podSubnet: "10.200.0.0/16"
nodes:
  - role: control-plane
  - role: worker
KIONF

  provisioner "local-exec" {
    command = "liqoctl install kind --cluster-name milan --kubeconfig './milan-config'"
  }
}
