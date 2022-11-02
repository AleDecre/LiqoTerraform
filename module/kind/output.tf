output "kubeconfig_path" {
  value = "${kind_cluster.default.kubeconfig_path}"
}

output "kubeconfig" {
  value = "${kind_cluster.default.kubeconfig}"
}