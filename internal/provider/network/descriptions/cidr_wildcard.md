Returns the wildcard mask for the given IPv4 CIDR. For example, `10.0.0.0/24` → `0.0.0.255`. IPv6 CIDRs return an error.

**Common uses:** generating Cisco IOS ACL entries, AWS network ACL wildcard fields, firewall rules that use wildcard mask notation instead of prefix length.