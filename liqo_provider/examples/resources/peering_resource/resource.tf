# Peer two clusters.
resource "liqo_peering" "peering" {

  cluster_id      = "<cluster_id>"
  cluster_name    = "<cluster_name>"
  cluster_authurl = "<auth-url>"
  cluster_token   = "<cluster_token>"

}
