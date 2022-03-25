package influx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

func resourceBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCreate,
		ReadContext:   resourceBucketRead,
		UpdateContext: resourceBucketUpdate,
		DeleteContext: resourceBucketDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dbrp_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"org_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the bucket",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the bucket",
			},
			"retention_days": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Retention time of the data in days",
			},
		},
	}
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("creating bucket with name: %s", d.Get("name").(string))

	api := getInfluxClientFromMetadata(m)
	org := getInfluxOrgFromMetadata(m)

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	bucket := domain.Bucket{
		OrgID:          &org,
		Name:           name,
		Description:    &description,
		RetentionRules: []domain.RetentionRule{},
	}

	ret := "inf"
	if r, ok := d.GetOk("retention_days"); ok {
		bucket.RetentionRules = append(bucket.RetentionRules, domain.RetentionRule{
			EverySeconds: int64(r.(int) * 24 * 60 * 60),
		})
		ret = fmt.Sprintf("%ddays", r.(int))
	}

	resp, err := api.BucketsAPI().CreateBucket(ctx, &bucket)
	if err != nil {
		return diag.Errorf("failed to create bucket: %v", err)
	}

	d.SetId(*resp.Id)

	body, err := json.Marshal(map[string]interface{}{
		"bucketID":         *resp.Id,
		"database":         name,
		"default":          true,
		"orgID":            *resp.OrgID,
		"retention_policy": ret,
	})
	if err != nil {
		return diag.Errorf("failed to create rp mapping body: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%sdbrps?orgID=%s", api.HTTPService().ServerAPIURL(), *resp.OrgID),
		bytes.NewReader(body))
	if err != nil {
		return diag.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", getInfluxTokenFromMetadata(m)))
	req.Header.Add("Content-type", "application/json")

	mappingResp, err := client.Do(req)
	if err != nil {
		cleanErr := resourceBucketDelete(ctx, d, m)
		if cleanErr != nil {
			return diag.Errorf("failed bucket cleanup: %v", cleanErr)
		}
		return diag.Errorf("failed during create rp mapping request: %v", err)
	}

	respBody, _ := ioutil.ReadAll(mappingResp.Body)

	if mappingResp.StatusCode != 201 {
		cleanErr := resourceBucketDelete(ctx, d, m)
		if cleanErr != nil {
			return diag.Errorf("failed bucket cleanup: %v", cleanErr)
		}
		return diag.Errorf("failed during create rp mapping request: %v", string(respBody))
	}

	return resourceBucketRead(ctx, d, m)
}

func resourceBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("reading bucket with id: %s", d.Id())

	api := getInfluxClientFromMetadata(m)

	bucket, err := api.BucketsAPI().FindBucketByID(ctx, d.Id())
	if err != nil && err.Error() != "not found: bucket not found" {
		return diag.Errorf("failed to get bucket: %v", err)
	}

	if bucket == nil {
		d.SetId("")
		return nil
	}

	rpID, err := getMappingID(ctx, d, m)
	if err != nil {
		return diag.Errorf("no database retention policy id found: %v", err)
	}

	_ = d.Set("name", bucket.Name)
	_ = d.Set("dbrp_id", rpID)
	_ = d.Set("description", bucket.Description)
	_ = d.Set("org_id", bucket.OrgID)

	d.Set("retention_days", 0)
	if len(bucket.RetentionRules) > 0 {
		d.Set("retention_days", bucket.RetentionRules[0].EverySeconds/24/60/60)
	}

	return nil
}

func resourceBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	api := getInfluxClientFromMetadata(m)
	org := getInfluxOrgFromMetadata(m)

	log.Printf("updating bucket id: %s", id)

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	orgID := d.Get("org_id").(string)
	dbrpID := d.Get("dbrp_id").(string)

	bucket := domain.Bucket{
		Id:             &id,
		OrgID:          &org,
		Name:           name,
		Description:    &description,
		RetentionRules: []domain.RetentionRule{},
	}

	ret := "inf"
	if r, ok := d.GetOk("retention_days"); ok {
		bucket.RetentionRules = append(bucket.RetentionRules, domain.RetentionRule{
			EverySeconds: int64(r.(int) * 24 * 60 * 60),
		})
		ret = fmt.Sprintf("%ddays", r.(int))
	}

	_, err := api.BucketsAPI().UpdateBucket(ctx, &bucket)
	if err != nil {
		return diag.Errorf("failed to update bucket: %v", err)
	}

	body, err := json.Marshal(map[string]interface{}{
		"retention_policy": ret,
	})
	if err != nil {
		return diag.Errorf("failed to update rp mapping body: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("PATCH",
		fmt.Sprintf("%sdbrps/%s?orgID=%s", api.HTTPService().ServerAPIURL(), dbrpID, orgID),
		bytes.NewReader(body))
	if err != nil {
		return diag.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", getInfluxTokenFromMetadata(m)))
	req.Header.Add("Content-type", "application/json")

	mappingResp, err := client.Do(req)
	if err != nil {
		return diag.Errorf("failed during update rp mapping request: %v", err)
	}

	respBody, _ := ioutil.ReadAll(mappingResp.Body)

	if mappingResp.StatusCode != 200 {
		return diag.Errorf("failed during update rp mapping request: %v", string(respBody))
	}

	return resourceBucketRead(ctx, d, m)
}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("deleting bucket id: %s", d.Id())

	api := getInfluxClientFromMetadata(m)

	err := api.BucketsAPI().DeleteBucketWithID(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete bucket: %v", err)
	}

	url := fmt.Sprintf("%sdbrps/%s?orgID=%s", api.HTTPService().ServerAPIURL(), d.Get("dbrp_id").(string), d.Get("org_id").(string))
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return diag.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", getInfluxTokenFromMetadata(m)))

	_, err = client.Do(req)
	if err != nil {
		return diag.Errorf("failed to delete rp mapping: %v", err)
	}

	return nil
}

func getMappingID(ctx context.Context, d *schema.ResourceData, m interface{}) (string, error) {
	api := getInfluxClientFromMetadata(m)

	url := fmt.Sprintf("%sdbrps/?orgID=%s&bucketID=%s", api.HTTPService().ServerAPIURL(), d.Get("org_id"), d.Id())
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", getInfluxTokenFromMetadata(m)))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed during create rp mapping request: %v", err)
	}

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed during create rp mapping request: %v", string(respBody))
	}

	respData := map[string]interface{}{}
	err = json.Unmarshal(respBody, &respData)
	if err != nil {
		return "", fmt.Errorf("unable to parse mapping json response: %v", err)
	}

	if content, ok := respData["content"]; ok {
		results := content.([]interface{})
		if len(results) > 0 {
			if id, ok := results[0].(map[string]interface{})["id"]; ok {
				return id.(string), nil
			}
		}
	}

	return "", fmt.Errorf("no mapping id found")
}
