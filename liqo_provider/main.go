package main

import (
	"context"

	"terraform-provider-test/liqo"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), liqo.New, providerserver.ServeOpts{
		Address: "liqo-provider/liqo/test",
	})
}
