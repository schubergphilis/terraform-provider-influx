package influx

import (
	"context"
	"log"

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
			"retention": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Retention time of the data in seconds",
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

	if r, ok := d.GetOk("retention"); ok {
		bucket.RetentionRules = append(bucket.RetentionRules, domain.RetentionRule{
			EverySeconds: r.(int),
		})
	}

	resp, err := api.BucketsAPI().CreateBucket(ctx, &bucket)
	if err != nil {
		return diag.Errorf("failed to create bucket: %v", err)
	}

	d.SetId(*resp.Id)

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

	_ = d.Set("name", bucket.Name)
	_ = d.Set("description", bucket.Description)

	d.Set("retention", 0)
	if len(bucket.RetentionRules) > 0 {
		d.Set("retention", bucket.RetentionRules[0].EverySeconds)
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

	bucket := domain.Bucket{
		Id:             &id,
		OrgID:          &org,
		Name:           name,
		Description:    &description,
		RetentionRules: []domain.RetentionRule{},
	}

	if r, ok := d.GetOk("retention"); ok {
		bucket.RetentionRules = append(bucket.RetentionRules, domain.RetentionRule{
			EverySeconds: r.(int),
		})
	}

	_, err := api.BucketsAPI().UpdateBucket(ctx, &bucket)
	if err != nil {
		return diag.Errorf("failed to create bucket: %v", err)
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

	return nil
}
