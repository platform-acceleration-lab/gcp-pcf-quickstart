// Allow open access between internal VMs for a PCF deployment
resource "google_compute_firewall" "cf-internal" {
  name    = "${var.env_name}-cf-internal"
  network = "${google_compute_network.pcf-network.name}"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
  }

  allow {
    protocol = "udp"
  }

  source_ranges = [
    "${google_compute_subnetwork.management-subnet.ip_cidr_range}",
    "${google_compute_subnetwork.unmanaged-subnet.ip_cidr_range}",
    "${google_compute_subnetwork.pas-subnet.ip_cidr_range}",
    "${google_compute_subnetwork.services-subnet.ip_cidr_range}",
    "${google_compute_subnetwork.dynamic-services-subnet.ip_cidr_range}",
  ]
}
