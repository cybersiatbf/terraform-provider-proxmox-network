package proxmox

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			"cybernordic-proxmox-network_interface": resourceProxmoxNetworkInterface(),
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
