package main

import (
	proxmox "github.com/cybersiatbf/terraform-provider-proxmox-network/internal/provider"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: proxmox.Provider,
	})
}
