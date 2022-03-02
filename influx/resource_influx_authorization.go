package influx

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

func resourceAuthorization() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAuthorizationCreate,
		ReadContext:   resourceAuthorizationRead,
		UpdateContext: resourceAuthorizationUpdate,
		DeleteContext: resourceAuthorizationDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"org_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"host": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the authorization",
			},
			"organization": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "active",
				Description: "Status of the token. Either active or inactive. defaults to active",
			},
			"permission": {
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Description: "List of permissions for the authorization",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "The id of the resource",
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "The type of the resource",
						},
						"id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "The id of the resource",
						},
					},
				},
			},
		},
	}
}

func resourceAuthorizationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("creating authorization with name: %s", d.Get("name").(string))

	api := getInfluxClientFromMetadata(m)
	org := getInfluxOrgFromMetadata(m)

	name := d.Get("name").(string)
	raw := d.Get("permission").(*schema.Set)
	permissions := []domain.Permission{}

	for _, p := range raw.List() {
		rawPermission := p.(map[string]interface{})

		resource := domain.Resource{
			Type:  domain.ResourceType(rawPermission["type"].(string)),
			OrgID: &org,
		}

		permission := domain.Permission{
			Action:   domain.PermissionActionRead,
			Resource: resource,
		}

		a, _ := rawPermission["action"].(string)
		if a == "write" {
			permission.Action = domain.PermissionActionWrite
		}

		if v, ok := rawPermission["id"].(string); ok && len(v) > 0 {
			permission.Resource.Id = &v
		}

		permissions = append(permissions, permission)
	}

	authorization := domain.Authorization{
		OrgID:       &org,
		Permissions: &permissions,
		AuthorizationUpdateRequest: domain.AuthorizationUpdateRequest{
			Description: &name,
		},
	}

	resp, err := api.AuthorizationsAPI().CreateAuthorization(ctx, &authorization)
	if err != nil {
		return diag.Errorf("failed to create authorization: %v", err)
	}

	d.SetId(*resp.Id)
	_ = d.Set("token", *resp.Token)

	return resourceAuthorizationRead(ctx, d, m)
}

func resourceAuthorizationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("reading authorization with id: %s", d.Id())

	api := getInfluxClientFromMetadata(m)

	authorizations, err := api.AuthorizationsAPI().GetAuthorizations(ctx)
	if err != nil {
		return diag.Errorf("failed to get authorization: %v", err)
	}

	var authorization *domain.Authorization

	for _, a := range *authorizations {
		if *a.Id == d.Id() {
			authorization = &a
			break
		}
	}

	if authorization == nil {
		d.SetId("")
		return nil
	}

	permissions := make([]map[string]string, len(*authorization.Permissions))
	for i, p := range *authorization.Permissions {
		permissions[i] = map[string]string{
			"action": string(p.Action),
			"type":   string(p.Resource.Type),
		}
		if p.Resource.Id != nil {
			permissions[i]["id"] = *p.Resource.Id
		}
	}

	_ = d.Set("host", api.ServerURL())
	_ = d.Set("name", authorization.Description)
	_ = d.Set("org_id", authorization.OrgID)
	_ = d.Set("organization", authorization.Org)
	_ = d.Set("permission", permissions)
	_ = d.Set("status", authorization.Status)

	return nil
}

func resourceAuthorizationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("updating authorization id: %s", d.Id())

	api := getInfluxClientFromMetadata(m)

	statusChange := d.HasChange("status")
	if statusChange {
		status := domain.AuthorizationUpdateRequestStatusActive
		if d.Get("status").(string) == "inactive" {
			status = domain.AuthorizationUpdateRequestStatusInactive
		}

		_, err := api.AuthorizationsAPI().UpdateAuthorizationStatusWithID(ctx, d.Id(), status)
		if err != nil {
			return diag.Errorf("failed to update authorization status: %v", err)
		}
	}

	return nil
}

func resourceAuthorizationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("deleting authorization id: %s", d.Id())

	api := getInfluxClientFromMetadata(m)

	err := api.AuthorizationsAPI().DeleteAuthorizationWithID(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete authorization: %v", err)
	}

	return nil
}
