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

module "kind" {

  for_each = {
    for cluster in var.clusters.clusters_list :
    cluster.name => cluster
  }

  source = "./module/kind"

  cluster      = each.value
  kind_version = var.kind_version

}


resource "null_resource" "cluster_peering" {

  for_each = {
    for index, cluster in var.clusters.clusters_list :
    cluster.name => cluster if index != 0 && var.clusters.peering
  }

  provisioner "local-exec" {

    command = "$(liqoctl generate peer-command --only-command --kubeconfig \"$KUBECONFIG_REMOTE\")"

    environment = {
      KUBECONFIG        = "${module.kind[var.clusters.clusters_list[0].name].kubeconfig_path}"
      KUBECONFIG_REMOTE = "${module.kind[each.value.name].kubeconfig_path}"
    }

  }
  /*
  provisioner "local-exec" {

    command = "$(liqoctl generate peer-command --only-command --kubeconfig \"$KUBECONFIG_REMOTE\")"

    environment = {
      KUBECONFIG        = "${module.kind["rome"] kubeconfig_path}"
      KUBECONFIG_REMOTE = "${module.kind[each.value.name].kubeconfig_path}"
    }

  }
*/
}

