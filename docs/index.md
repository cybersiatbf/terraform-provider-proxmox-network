provider "cybernordic-proxmox-network" {
  api_url   = var.proxmox_url
  api_token = "${var.proxmox_id}=${var.proxmox_secret}"
}

resource "proxmox-network_bridge" "bridge" {
  node              = "prox"
  type              = "OVSBridge"
  iface             = "vmbr500"
  cidr              = "10.1.0.0/24"
  comments          = "OVSBridge"
}