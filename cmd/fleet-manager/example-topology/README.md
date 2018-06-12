# Example topology for Fleet Manager

This example toplogy contains 2 large regions (NYC and SJC) and a small region
(SYD). Each region has 3 VLANS:
- `Production`: for products serving customers
- `Infrastructure`: for internal infrastructure services
- `Egress`: for VMs which have Internet egress access via a NAT gateway

While the large regions have the `Production` and `Infrastructure` subnets
segmented per rack, the smaller SYD region has all subnets covering the entire
region.
