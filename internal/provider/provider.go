package proxmox

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	inflightOps int32
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROXMOX_API_URL", nil),
				Description: "The URL of the Proxmox API.",
			},
			"api_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROXMOX_API_TOKEN", nil),
				Description: "The API token for accessing Proxmox.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"proxmox-network_interface": resourceProxmoxNetworkInterface(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &ClientConfig{
		APIURL:   d.Get("api_url").(string),
		APIToken: d.Get("api_token").(string),
	}
	return config, nil
}

func maybeApplyConf(d *schema.ResourceData, meta interface{}) {
	if atomic.AddInt32(&inflightOps, -1) == 0 {
		client := meta.(*ClientConfig)
		node := d.Get("node").(string)
		endpoint := fmt.Sprintf("nodes/%s/network", node)
		data := map[string]interface{}{}

		_, err := client.doRequest("PUT", endpoint, data)
		if err != nil {
			log.Printf("[ERROR] Failed to apply network config: %v", err)
		}
	}
}
