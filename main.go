package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/schubergphilis/terraform-provider-influx/influx"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: influx.New,
	})
}
