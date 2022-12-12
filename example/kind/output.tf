output "kubeconfig_path_rome" {
  value = "${module.kind["rome"].kubeconfig_path}"
}

output "kubeconfig_path_milan" {
  value = "${module.kind["milan"].kubeconfig_path}"
}