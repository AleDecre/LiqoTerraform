terraform {
  required_providers {
    hashicups = {
      source = "liqo-provider/liqo/test"
    }
  }
}

provider "hashicups" {}

data "hashicups_coffees" "example" {}
