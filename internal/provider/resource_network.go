package proxmox

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceProxmoxNetworkInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceProxmoxNetworkInterfaceCreate,
		Read:   resourceProxmoxNetworkInterfaceRead,
		Update: resourceProxmoxNetworkInterfaceUpdate,
		Delete: resourceProxmoxNetworkInterfaceDelete,

		Schema: map[string]*schema.Schema{
			"node":  {Type: schema.TypeString, Required: true},
			"iface": {Type: schema.TypeString, Required: true},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"OVSBridge", "OVSIntPort", "bridge", "vlan"}, false),
			},
			"cidr":              {Type: schema.TypeString, Optional: true},
			"autostart":         {Type: schema.TypeBool, Optional: true, Default: true},
			"bridge_ports":      {Type: schema.TypeString, Optional: true},
			"bridge_vlan_aware": {Type: schema.TypeBool, Optional: true},
			"ovs_bridge":        {Type: schema.TypeString, Optional: true},
			"ovs_tag":           {Type: schema.TypeInt, Optional: true},
			"address":           {Type: schema.TypeString, Optional: true},
			"netmask":           {Type: schema.TypeString, Optional: true},
			"comments":          {Type: schema.TypeString, Optional: true},
		},
	}
}

func resourceProxmoxNetworkInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	atomic.AddInt32(&inflightOps, 1)
	defer maybeApplyConf(d, meta)

	client := meta.(*ClientConfig)

	data := map[string]interface{}{
		"iface":     d.Get("iface").(string),
		"type":      d.Get("type").(string),
		"autostart": d.Get("autostart").(bool),
	}

	if v, ok := d.GetOk("cidr"); ok {
		data["cidr"] = v.(string)
	}
	if v, ok := d.GetOk("bridge_ports"); ok {
		data["bridge_ports"] = v.(string)
	}
	if v, ok := d.GetOk("bridge_vlan_aware"); ok {
		data["bridge_vlan_aware"] = v.(bool)
	}
	if v, ok := d.GetOk("ovs_bridge"); ok {
		data["ovs_bridge"] = v.(string)
	}
	if v, ok := d.GetOk("ovs_tag"); ok {
		data["ovs_tag"] = v.(int)
	}
	if v, ok := d.GetOk("address"); ok {
		data["address"] = v.(string)
	}
	if v, ok := d.GetOk("netmask"); ok {
		data["netmask"] = v.(string)
	}
	if v, ok := d.GetOk("comments"); ok {
		data["comments"] = v.(string)
	}

	node := d.Get("node").(string)
	_, err := client.doRequest("POST", fmt.Sprintf("nodes/%s/network", node), data)
	if err != nil {
		return fmt.Errorf("failed to create network interface: %v", err)
	}

	d.SetId(fmt.Sprintf("%s/%s", node, d.Get("iface").(string)))
	// applyNetworkConf(d, meta)
	return resourceProxmoxNetworkInterfaceRead(d, meta)
}

func resourceProxmoxNetworkInterfaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ClientConfig)
	node := d.Get("node").(string)
	iface := d.Get("iface").(string)

	endpoint := fmt.Sprintf("nodes/%s/network/%s", node, iface)
	resp, err := client.doRequest("GET", endpoint, nil)
	if err != nil {
		if strings.Contains(err.Error(), "API error 404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to read interface: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return fmt.Errorf("failed to parse interface response: %v", err)
	}

	for _, key := range []string{
		"iface", "type", "cidr", "autostart", "bridge_ports",
		"bridge_vlan_aware", "ovs_bridge", "ovs_tag", "address",
		"netmask", "comments",
	} {
		if v, ok := d.GetOk(key); ok {
			data[key] = v
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", node, iface))
	return nil
}

func resourceProxmoxNetworkInterfaceUpdate(d *schema.ResourceData, meta interface{}) error {
	atomic.AddInt32(&inflightOps, 1)
	defer maybeApplyConf(d, meta)

	client := meta.(*ClientConfig)

	node := d.Get("node").(string)
	iface := d.Get("iface").(string)
	endpoint := fmt.Sprintf("nodes/%s/network/%s", node, iface)

	data := map[string]interface{}{
		"type":  d.Get("type").(string),
		"iface": d.Get("iface").(string),
	}

	for _, key := range []string{
		"cidr", "autostart", "bridge_ports",
		"bridge_vlan_aware", "ovs_bridge", "ovs_tag",
		"address", "netmask", "comments",
	} {
		if v, ok := d.GetOk(key); ok {
			data[key] = v
		}
	}

	_, err := client.doRequest("PUT", endpoint, data)
	if err != nil {
		return fmt.Errorf("failed to update interface: %v", err)
	}

	// applyNetworkConf(d, meta)
	return resourceProxmoxNetworkInterfaceRead(d, meta)
}

func resourceProxmoxNetworkInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	atomic.AddInt32(&inflightOps, 1)
	defer maybeApplyConf(d, meta)

	client := meta.(*ClientConfig)

	node := d.Get("node").(string)
	iface := d.Get("iface").(string)
	endpoint := fmt.Sprintf("nodes/%s/network/%s", node, iface)

	_, err := client.doRequest("DELETE", endpoint, nil)
	if err != nil {
		if strings.Contains(err.Error(), "API error 404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to read interface: %v", err)
	}

	// applyNetworkConf(d, meta)

	// Wait and confirm deletion
	for i := 0; i < 5; i++ {
		resp, err := client.doRequest("GET", fmt.Sprintf("nodes/%s/network", node), nil)
		if err != nil {
			break
		}
		var interfaces []map[string]interface{}
		json.Unmarshal(resp, &interfaces)

		stillExists := false
		for _, net := range interfaces {
			if net["iface"] == iface {
				stillExists = true
				break
			}
		}

		if !stillExists {
			d.SetId("")
			return nil
		}

		time.Sleep(2 * time.Second)
		// applyNetworkConf(d, meta)
	}

	d.SetId("")
	return fmt.Errorf("interface %s deletion could not be verified", iface)
}

func applyNetworkConf(d *schema.ResourceData, meta interface{}) {
	client := meta.(*ClientConfig)

	node := d.Get("node").(string)

	endpoint := fmt.Sprintf("nodes/%s/network", node)

	data := map[string]interface{}{}

	client.doRequest("PUT", endpoint, data)
}
