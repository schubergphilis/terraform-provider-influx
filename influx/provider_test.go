package influx

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = New()
	testAccProviders = map[string]*schema.Provider{
		"influx": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = New()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("O365_ACL_GUID"); v == "" {
		t.Fatal("O365_ACL_GUID must be set for acceptance tests")
	}
	if v := os.Getenv("O365_ALIAS"); v == "" {
		t.Fatal("O365_ALIAS must be set for acceptance tests")
	}
	if v := os.Getenv("O365_EXOAPI_ENDPOINT"); v == "" {
		t.Fatal("O365_EXOAPI_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("O365_GROUP_ID"); v == "" {
		t.Fatal("O365_GROUP_ID must be set for acceptance tests")
	}
	if v := os.Getenv("O365_SECRET_CODE"); v == "" {
		t.Fatal("O365_SECRET_CODE must be set for acceptance tests")
	}
}
