package influx

import (
	"context"
	"log"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Config contains our provider schema values and Okta clients
type Config struct {
	url          string
	token        string
	org          string
	logLevel     int
	influxClient influxdb2.Client
}

// influxClient configures and returns a fully initialized InfluxClient.
func (c *Config) loadAndValidate() error {
	log.Print("Building InfluxClient")

	options := influxdb2.DefaultOptions()
	options.SetLogLevel(uint(c.logLevel))

	client := influxdb2.NewClientWithOptions(c.url, c.token, options)

	res, err := client.OrganizationsAPI().GetOrganizations(context.Background())
	if err != nil {
		return err
	}

	orgs := *res

	if len(orgs) > 0 {
		c.org = *orgs[0].Id
	}

	c.influxClient = client

	return nil
}
