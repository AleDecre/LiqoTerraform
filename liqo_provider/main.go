package main

import (
	"context"

	"terraform-provider-liqo/liqo"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name liqo

func main() {
	providerserver.Serve(context.Background(), liqo.New, providerserver.ServeOpts{
		Address: "liqo-provider/liqo/liqo",
	})
}
