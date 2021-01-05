package influx

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func getInfluxClientFromMetadata(meta interface{}) influxdb2.Client {
	return meta.(*Config).influxClient
}

func getInfluxOrgFromMetadata(meta interface{}) string {
	return meta.(*Config).org
}

func getInfluxTokenFromMetadata(meta interface{}) string {
	return meta.(*Config).token
}
