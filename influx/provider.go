// Package influx implements a custom influx Terraform provider.
package influx

import (
	"context"
	"log"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// New returns a schema.Provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("INFLUX_URL", ""),
				Description: "The InfluxDB API URL",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("INFLUX_TOKEN", ""),
				Description: "The InfluxDBAPI Token granting privileges to Influx API.",
			},
			"log_level": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     int(hclog.Error),
				Description: "providers log level. Minimum is 1 (TRACE), and maximum is 5 (ERROR)",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"influx_bucket":        resourceBucket(),
			"influx_authorization": resourceAuthorization(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	log.Printf("Initializing Influx client")

	config := Config{
		url:      d.Get("url").(string),
		token:    d.Get("token").(string),
		logLevel: d.Get("log_level").(int),
	}
	if err := config.loadAndValidate(); err != nil {
		return nil, diag.Errorf("Error initializing the Influx API client: %v", err)
	}
	return &config, nil
}
