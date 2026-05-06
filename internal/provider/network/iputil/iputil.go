package iputil

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"

	"go4.org/netipx"
)

// ParsePrefix parses a CIDR string and returns the masked (normalized) prefix.
// e.g. "10.0.0.1/24" → 10.0.0.0/24
func ParsePrefix(s string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid CIDR %q: %w", s, err)
	}
	return p.Masked(), nil
}

// ParseAddr parses an IP address string. IPv4-mapped IPv6 addresses are
// unmapped to their native IPv4 form.
func ParseAddr(s string) (netip.Addr, error) {
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid IP %q: %w", s, err)
	}
	return a.Unmap(), nil
}

// LastIP returns the last address in a prefix (all host bits set).
func LastIP(p netip.Prefix) netip.Addr {
	return netipx.RangeOfPrefix(p).To()
}

// prefixesToStrings converts a slice of netip.Prefix to a slice of strings.
func prefixesToStrings(ps []netip.Prefix) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	return out
}

// buildSet parses a list of CIDR strings into an IPSetBuilder.
func buildSet(cidrs []string) (netipx.IPSetBuilder, error) {
	var b netipx.IPSetBuilder
	for _, c := range cidrs {
		p, err := ParsePrefix(c)
		if err != nil {
			return b, err
		}
		b.AddPrefix(p)
	}
	return b, nil
}

// ---- CIDR set operations ----

// MergeCIDRs aggregates a list of CIDR strings into the smallest equivalent set.
func MergeCIDRs(cidrs []string) ([]string, error) {
	b, err := buildSet(cidrs)
	if err != nil {
		return nil, err
	}
	s, err := b.IPSet()
	if err != nil {
		return nil, err
	}
	return prefixesToStrings(s.Prefixes()), nil
}

// SubtractCIDRs removes all exclude CIDRs from the input list.
func SubtractCIDRs(inputs, excludes []string) ([]string, error) {
	b, err := buildSet(inputs)
	if err != nil {
		return nil, err
	}
	for _, c := range excludes {
		p, err := ParsePrefix(c)
		if err != nil {
			return nil, err
		}
		b.RemovePrefix(p)
	}
	s, err := b.IPSet()
	if err != nil {
		return nil, err
	}
	return prefixesToStrings(s.Prefixes()), nil
}

// IntersectCIDRs returns the set of CIDRs representing the intersection of two lists.
func IntersectCIDRs(as, bs []string) ([]string, error) {
	aSet, err := buildSet(as)
	if err != nil {
		return nil, err
	}
	a, err := aSet.IPSet()
	if err != nil {
		return nil, err
	}
	bSet, err := buildSet(bs)
	if err != nil {
		return nil, err
	}
	bSet.Intersect(a)
	result, err := bSet.IPSet()
	if err != nil {
		return nil, err
	}
	return prefixesToStrings(result.Prefixes()), nil
}

const maxExpand = 65536

// ExpandCIDR returns every individual IP address in the given CIDR.
// Returns an error if the CIDR contains more than 65536 addresses.
func ExpandCIDR(cidr string) ([]string, error) {
	p, err := ParsePrefix(cidr)
	if err != nil {
		return nil, err
	}
	hostBits := p.Addr().BitLen() - p.Bits()
	if hostBits >= 17 {
		count := 1 << uint(hostBits)
		return nil, fmt.Errorf("CIDR %s would expand to %d addresses, exceeding the limit of %d", cidr, count, maxExpand)
	}
	count := 1 << uint(hostBits)
	if count > maxExpand {
		return nil, fmt.Errorf("CIDR %s would expand to %d addresses, exceeding the limit of %d", cidr, count, maxExpand)
	}
	result := make([]string, 0, count)
	cur := p.Addr()
	for i := 0; i < count; i++ {
		result = append(result, cur.String())
		cur = cur.Next()
	}
	return result, nil
}

// RangeToCIDRs converts an inclusive IP range [firstStr, lastStr] into the
// minimal list of CIDRs that exactly covers it.
func RangeToCIDRs(firstStr, lastStr string) ([]string, error) {
	first, err := ParseAddr(firstStr)
	if err != nil {
		return nil, err
	}
	last, err := ParseAddr(lastStr)
	if err != nil {
		return nil, err
	}
	if first.Compare(last) > 0 {
		return nil, fmt.Errorf("first IP %s is after last IP %s", firstStr, lastStr)
	}
	if first.Is4() != last.Is4() {
		return nil, fmt.Errorf("first and last IPs must be the same address family")
	}
	r := netipx.IPRangeFrom(first, last)
	if !r.Valid() {
		return nil, fmt.Errorf("invalid IP range %s–%s", firstStr, lastStr)
	}
	return prefixesToStrings(r.Prefixes()), nil
}

// ---- Query functions ----

// IPInCIDR reports whether ip is contained in cidr.
func IPInCIDR(ipStr, cidrStr string) (bool, error) {
	ip, err := ParseAddr(ipStr)
	if err != nil {
		return false, err
	}
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return false, err
	}
	return p.Contains(ip), nil
}

// IPInCIDRs returns all CIDRs in cidrStrs that contain ip.
func CIDRsContainingIP(ipStr string, cidrStrs []string) ([]string, error) {
	ip, err := ParseAddr(ipStr)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, c := range cidrStrs {
		p, err := ParsePrefix(c)
		if err != nil {
			return nil, err
		}
		if p.Contains(ip) {
			result = append(result, p.String())
		}
	}
	return result, nil
}

// CIDRContains reports whether cidr fully contains other, which may be a bare IP or a CIDR.
func CIDRContains(cidrStr, otherStr string) (bool, error) {
	outer, err := ParsePrefix(cidrStr)
	if err != nil {
		return false, err
	}
	var ob netipx.IPSetBuilder
	ob.AddPrefix(outer)
	outerSet, err := ob.IPSet()
	if err != nil {
		return false, err
	}
	// Try as a bare IP first.
	if ip, err2 := netip.ParseAddr(otherStr); err2 == nil {
		return outerSet.Contains(ip.Unmap()), nil
	}
	inner, err := ParsePrefix(otherStr)
	if err != nil {
		return false, fmt.Errorf("other must be a valid IP or CIDR: %w", err)
	}
	return outerSet.ContainsPrefix(inner), nil
}

// CIDROverlaps reports whether two CIDRs share any addresses.
func CIDROverlaps(aStr, bStr string) (bool, error) {
	a, err := ParsePrefix(aStr)
	if err != nil {
		return false, err
	}
	b, err := ParsePrefix(bStr)
	if err != nil {
		return false, err
	}
	return a.Overlaps(b), nil
}

// CIDRsOverlapAny returns true if any CIDR in list a overlaps with any CIDR in list b.
func CIDRsOverlapAny(as, bs []string) (bool, error) {
	aSet, err := buildSet(as)
	if err != nil {
		return false, err
	}
	a, err := aSet.IPSet()
	if err != nil {
		return false, err
	}
	for _, c := range bs {
		p, err := ParsePrefix(c)
		if err != nil {
			return false, err
		}
		if a.OverlapsPrefix(p) {
			return true, nil
		}
	}
	return false, nil
}

// ---- Info functions ----

// CIDRHostCount returns the total number of addresses in the CIDR (including
// network and broadcast for IPv4). For very large IPv6 prefixes the result is
// capped at math.MaxInt64.
func CIDRHostCount(cidrStr string) (int64, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return 0, err
	}
	hostBits := p.Addr().BitLen() - p.Bits()
	if hostBits >= 63 {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(1) << uint(hostBits), nil
}

// CIDRFirstIP returns the network address of the prefix (first IP).
func CIDRFirstIP(cidrStr string) (string, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return "", err
	}
	return p.Addr().String(), nil
}

// CIDRLastIP returns the last address in the prefix.
func CIDRLastIP(cidrStr string) (string, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return "", err
	}
	return LastIP(p).String(), nil
}

// IPVersion returns 4 for IPv4 addresses and 6 for IPv6 addresses.
func IPVersion(ipStr string) (int64, error) {
	ip, err := ParseAddr(ipStr)
	if err != nil {
		return 0, err
	}
	if ip.Is4() {
		return 4, nil
	}
	return 6, nil
}

// CIDRVersion returns 4 for IPv4 prefixes and 6 for IPv6 prefixes.
func CIDRVersion(cidrStr string) (int64, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return 0, err
	}
	if p.Addr().Is4() {
		return 4, nil
	}
	return 6, nil
}

// privateRanges includes RFC1918, loopback, link-local, RFC6598, and IPv6 ULA/loopback/link-local.
var privateRanges []netip.Prefix

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",    // IPv4 loopback
		"169.254.0.0/16", // IPv4 link-local
		"100.64.0.0/10",  // RFC6598 shared/CGNAT
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local (RFC4193)
		"fe80::/10",      // IPv6 link-local
	} {
		privateRanges = append(privateRanges, netip.MustParsePrefix(cidr))
	}
}

// IPIsPrivate reports whether ip is a private, loopback, or link-local address.
func IPIsPrivate(ipStr string) (bool, error) {
	ip, err := ParseAddr(ipStr)
	if err != nil {
		return false, err
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

// CIDRIsPrivate reports whether the entire CIDR falls within a single private range.
func CIDRIsPrivate(cidrStr string) (bool, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return false, err
	}
	first := p.Addr()
	last := LastIP(p)
	for _, r := range privateRanges {
		if r.Contains(first) && r.Contains(last) {
			return true, nil
		}
	}
	return false, nil
}

// FilterCIDRsByVersion returns only the CIDRs from the list that match the given IP version (4 or 6).
func FilterCIDRsByVersion(cidrStrs []string, version int64) ([]string, error) {
	if version != 4 && version != 6 {
		return nil, fmt.Errorf("version must be 4 or 6, got %d", version)
	}
	result := []string{}
	for _, c := range cidrStrs {
		p, err := ParsePrefix(c)
		if err != nil {
			return nil, err
		}
		is4 := p.Addr().Is4()
		if (version == 4 && is4) || (version == 6 && !is4) {
			result = append(result, p.String())
		}
	}
	return result, nil
}

// ---- General additions ----

// IPAdd returns the IP address offset by n (positive or negative).
// Returns an error if the result would overflow the address space.
func IPAdd(ipStr string, n int64) (string, error) {
	ip, err := ParseAddr(ipStr)
	if err != nil {
		return "", err
	}
	if ip.Is4() {
		a4 := ip.As4()
		cur := binary.BigEndian.Uint32(a4[:])
		if n >= 0 {
			if uint64(cur)+uint64(n) > 0xFFFFFFFF {
				return "", fmt.Errorf("ip_add: result overflows IPv4 address space")
			}
			binary.BigEndian.PutUint32(a4[:], cur+uint32(n))
		} else {
			abs := uint64(-n)
			if abs > uint64(cur) {
				return "", fmt.Errorf("ip_add: result underflows IPv4 address space")
			}
			binary.BigEndian.PutUint32(a4[:], cur-uint32(abs))
		}
		return netip.AddrFrom4(a4).String(), nil
	}
	raw := ip.As16()
	hi := binary.BigEndian.Uint64(raw[:8])
	lo := binary.BigEndian.Uint64(raw[8:])
	if n >= 0 {
		newLo := lo + uint64(n)
		carry := uint64(0)
		if newLo < lo {
			carry = 1
		}
		newHi := hi + carry
		if newHi < hi {
			return "", fmt.Errorf("ip_add: result overflows IPv6 address space")
		}
		binary.BigEndian.PutUint64(raw[:8], newHi)
		binary.BigEndian.PutUint64(raw[8:], newLo)
	} else {
		abs := uint64(-n)
		newLo := lo - abs
		borrow := uint64(0)
		if newLo > lo {
			borrow = 1
		}
		if borrow > hi {
			return "", fmt.Errorf("ip_add: result underflows IPv6 address space")
		}
		binary.BigEndian.PutUint64(raw[:8], hi-borrow)
		binary.BigEndian.PutUint64(raw[8:], newLo)
	}
	return netip.AddrFrom16(raw).String(), nil
}

// IPSubtract returns the signed integer distance a - b (i.e. how many addresses
// separate them). For IPv4 the result always fits in int64. For IPv6, returns
// an error if the high 64 bits of the two addresses differ or if the low-64-bit
// difference exceeds int64 range — in practice this only occurs when subtracting
// widely separated IPv6 addresses.
func IPSubtract(aStr, bStr string) (int64, error) {
	a, err := ParseAddr(aStr)
	if err != nil {
		return 0, err
	}
	b, err := ParseAddr(bStr)
	if err != nil {
		return 0, err
	}
	if a.Is4() != b.Is4() {
		return 0, fmt.Errorf("ip_subtract: addresses must be the same family (%s vs %s)", aStr, bStr)
	}
	if a.Is4() {
		a4, b4 := a.As4(), b.As4()
		an := int64(binary.BigEndian.Uint32(a4[:]))
		bn := int64(binary.BigEndian.Uint32(b4[:]))
		return an - bn, nil
	}
	// IPv6: split into high and low 64-bit halves.
	aRaw, bRaw := a.As16(), b.As16()
	aHi := binary.BigEndian.Uint64(aRaw[:8])
	aLo := binary.BigEndian.Uint64(aRaw[8:])
	bHi := binary.BigEndian.Uint64(bRaw[:8])
	bLo := binary.BigEndian.Uint64(bRaw[8:])
	if aHi != bHi {
		return 0, fmt.Errorf("ip_subtract: result exceeds int64 range (high 64 bits differ)")
	}
	const maxInt64 uint64 = 1<<63 - 1
	if aLo >= bLo {
		diff := aLo - bLo
		if diff > maxInt64 {
			return 0, fmt.Errorf("ip_subtract: result exceeds int64 range")
		}
		return int64(diff), nil
	}
	diff := bLo - aLo
	if diff > maxInt64+1 {
		return 0, fmt.Errorf("ip_subtract: result exceeds int64 range")
	}
	return -int64(diff), nil
}

const maxEnumerate = 65536

// EnumerateCIDR returns all sub-CIDRs of size (parent prefix length + newbits) within the given CIDR.
func EnumerateCIDR(cidrStr string, newbits int64) ([]string, error) {
	if newbits <= 0 {
		return nil, fmt.Errorf("newbits must be positive, got %d", newbits)
	}
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return nil, err
	}
	maxBits := p.Addr().BitLen()
	newPrefixLen := p.Bits() + int(newbits)
	if newPrefixLen > maxBits {
		return nil, fmt.Errorf("newbits %d would exceed maximum prefix length %d for this address family", newbits, maxBits)
	}
	count := int64(1) << uint(newbits)
	if count > maxEnumerate {
		return nil, fmt.Errorf("cidr_enumerate would produce %d subnets, exceeding the limit of %d", count, maxEnumerate)
	}
	result := make([]string, 0, int(count))
	cur := p.Addr()
	for i := int64(0); i < count; i++ {
		subnet := netip.PrefixFrom(cur, newPrefixLen).Masked()
		result = append(result, subnet.String())
		next := LastIP(subnet).Next()
		if !next.IsValid() {
			break
		}
		cur = next
	}
	return result, nil
}

// CIDRUsableHostCount returns the number of usable host addresses in a CIDR.
// For IPv4, this is host_count - 2 (subtracting network and broadcast addresses),
// with special cases: /31 = 2 (point-to-point, RFC 3021), /32 = 1 (host route).
// For IPv6, all addresses are usable so this equals CIDRHostCount.
func CIDRUsableHostCount(cidrStr string) (int64, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return 0, err
	}
	hostBits := p.Addr().BitLen() - p.Bits()
	if hostBits >= 63 {
		return int64(^uint64(0) >> 1), nil
	}
	total := int64(1) << uint(hostBits)
	if !p.Addr().Is4() {
		return total, nil
	}
	// IPv4 special cases
	switch p.Bits() {
	case 32:
		return 1, nil // host route
	case 31:
		return 2, nil // point-to-point (RFC 3021)
	default:
		return total - 2, nil
	}
}

// CIDRsAreDisjoint returns true if no two CIDRs in the list overlap each other.
func CIDRsAreDisjoint(cidrStrs []string) (bool, error) {
	// Build an IPSet from the list. If any prefixes overlap, the set will have
	// fewer addresses than the naive sum — but the simplest check is: build each
	// prefix's set individually and test against the running union.
	var union netipx.IPSetBuilder
	for _, c := range cidrStrs {
		p, err := ParsePrefix(c)
		if err != nil {
			return false, err
		}
		u, err := union.IPSet()
		if err != nil {
			return false, err
		}
		if u.OverlapsPrefix(p) {
			return false, nil
		}
		union.AddPrefix(p)
	}
	return true, nil
}

// CIDRPrefixLength returns the prefix length (the /N) of a CIDR.
func CIDRPrefixLength(cidrStr string) (int64, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return 0, err
	}
	return int64(p.Bits()), nil
}

// CIDRWildcard returns the wildcard mask (inverse of subnet mask) for an IPv4 CIDR.
func CIDRWildcard(cidrStr string) (string, error) {
	p, err := ParsePrefix(cidrStr)
	if err != nil {
		return "", err
	}
	if !p.Addr().Is4() {
		return "", fmt.Errorf("cidr_wildcard is only defined for IPv4 CIDRs, got %q", cidrStr)
	}
	hostBits := 32 - p.Bits()
	mask := (uint32(1) << uint(hostBits)) - 1
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], mask)
	return netip.AddrFrom4(b).String(), nil
}

// FindFreeCIDR returns the first available prefix of the given length within
// the pool after removing usedCIDRs. Returns nil if no prefix of that size is available.
func FindFreeCIDR(poolCIDRs, usedCIDRs []string, prefixLen int64) (*string, error) {
	if prefixLen < 0 || prefixLen > 128 {
		return nil, fmt.Errorf("prefix_len must be between 0 and 128, got %d", prefixLen)
	}
	b, err := buildSet(poolCIDRs)
	if err != nil {
		return nil, err
	}
	for _, c := range usedCIDRs {
		p, err := ParsePrefix(c)
		if err != nil {
			return nil, err
		}
		b.RemovePrefix(p)
	}
	available, err := b.IPSet()
	if err != nil {
		return nil, err
	}
	free, _, ok := available.RemoveFreePrefix(uint8(prefixLen))
	if !ok {
		return nil, nil
	}
	s := free.String()
	return &s, nil
}

// ---- NAT64 (RFC 6052) ----

var nat64ValidPrefixLengths = map[int]bool{32: true, 40: true, 48: true, 56: true, 64: true, 96: true}

// nat64PlaceIPv4 embeds a 4-byte IPv4 address into a 16-byte IPv6 raw buffer
// at the positions defined by RFC 6052 for the given prefix length.
// Byte index 8 (the u-octet) is always zeroed.
func nat64PlaceIPv4(raw *[16]byte, v4 [4]byte, pl int) {
	raw[8] = 0
	switch pl {
	case 32:
		raw[4], raw[5], raw[6], raw[7] = v4[0], v4[1], v4[2], 0
		raw[8] = v4[3]
	case 40:
		raw[5], raw[6], raw[7] = v4[0], v4[1], v4[2]
		raw[8] = 0
		raw[9] = v4[3]
	case 48:
		raw[6], raw[7] = v4[0], v4[1]
		raw[8] = 0
		raw[9], raw[10] = v4[2], v4[3]
	case 56:
		raw[7] = v4[0]
		raw[8] = 0
		raw[9], raw[10], raw[11] = v4[1], v4[2], v4[3]
	case 64:
		raw[8] = 0
		raw[9], raw[10], raw[11], raw[12] = v4[0], v4[1], v4[2], v4[3]
	case 96:
		raw[12], raw[13], raw[14], raw[15] = v4[0], v4[1], v4[2], v4[3]
	}
}

// nat64ExtractIPv4 reads the 4 IPv4 bytes from a 16-byte IPv6 raw buffer
// according to RFC 6052 for the given prefix length.
func nat64ExtractIPv4(raw [16]byte, pl int) [4]byte {
	var v4 [4]byte
	switch pl {
	case 32:
		v4[0], v4[1], v4[2], v4[3] = raw[4], raw[5], raw[6], raw[8]
	case 40:
		v4[0], v4[1], v4[2], v4[3] = raw[5], raw[6], raw[7], raw[9]
	case 48:
		v4[0], v4[1], v4[2], v4[3] = raw[6], raw[7], raw[9], raw[10]
	case 56:
		v4[0], v4[1], v4[2], v4[3] = raw[7], raw[9], raw[10], raw[11]
	case 64:
		v4[0], v4[1], v4[2], v4[3] = raw[9], raw[10], raw[11], raw[12]
	case 96:
		v4[0], v4[1], v4[2], v4[3] = raw[12], raw[13], raw[14], raw[15]
	}
	return v4
}

// NAT64PrefixValid reports whether prefix is a valid NAT64 prefix per RFC 6052.
func NAT64PrefixValid(prefixStr string) (bool, error) {
	p, err := netip.ParsePrefix(prefixStr)
	if err != nil {
		return false, fmt.Errorf("invalid prefix %q: %w", prefixStr, err)
	}
	if p.Addr().Is4() {
		return false, nil
	}
	if !nat64ValidPrefixLengths[p.Bits()] {
		return false, nil
	}
	// Check the u-octet (bits 64-71, byte 8) on the ORIGINAL (unmasked) address.
	// For prefix lengths ≤ /64, byte 8 lies in the host portion and would be zeroed
	// by .Masked(), hiding misconfigured prefixes. We want to reject those.
	raw := p.Addr().As16()
	if raw[8] != 0 {
		return false, nil
	}
	return true, nil
}

// NAT64Synthesize produces the IPv6 address that represents ipv4 under the given
// NAT64 prefix, following RFC 6052 Section 2.2.
func NAT64Synthesize(ipv4Str, prefixStr string, useMixed bool) (string, error) {
	ip, err := ParseAddr(ipv4Str)
	if err != nil {
		return "", err
	}
	if !ip.Is4() {
		return "", fmt.Errorf("expected IPv4 address, got %q", ipv4Str)
	}
	p, err := netip.ParsePrefix(prefixStr)
	if err != nil {
		return "", fmt.Errorf("invalid NAT64 prefix %q: %w", prefixStr, err)
	}
	p = p.Masked()
	if p.Addr().Is4() {
		return "", fmt.Errorf("NAT64 prefix must be IPv6, got %q", prefixStr)
	}
	if !nat64ValidPrefixLengths[p.Bits()] {
		return "", fmt.Errorf("NAT64 prefix length must be 32, 40, 48, 56, 64, or 96; got /%d", p.Bits())
	}
	raw := p.Addr().As16()
	nat64PlaceIPv4(&raw, ip.As4(), p.Bits())
	if useMixed {
		return ipToMixedNotation(raw), nil
	}
	return netip.AddrFrom16(raw).String(), nil
}

// NAT64Extract extracts the embedded IPv4 address from a NAT64 IPv6 address.
// NAT64Extract recovers the IPv4 address embedded in a NAT64 IPv6 address.
//
// With no prefix (nat64PrefixStr == ""): extracts the last 32 bits of the IPv6
// address as a dotted-decimal IPv4 address. This is correct for the common /96
// case (including the Well-Known Prefix 64:ff9b::/96) and requires no knowledge
// of which prefix was used.
//
// With a prefix: uses the RFC 6052 byte layout for that prefix length, which is
// needed for /32–/64 prefixes where the IPv4 bytes don't sit in the last 32 bits.
func NAT64Extract(ipv6Str, nat64PrefixStr string) (string, error) {
	ip, err := ParseAddr(ipv6Str)
	if err != nil {
		return "", err
	}
	if ip.Is4() {
		return "", fmt.Errorf("expected IPv6 address, got %q", ipv6Str)
	}
	// No prefix supplied: extract the last 32 bits directly (/96 behaviour).
	if nat64PrefixStr == "" {
		raw := ip.As16()
		return netip.AddrFrom4([4]byte{raw[12], raw[13], raw[14], raw[15]}).String(), nil
	}
	p, err := netip.ParsePrefix(nat64PrefixStr)
	if err != nil {
		return "", fmt.Errorf("invalid NAT64 prefix %q: %w", nat64PrefixStr, err)
	}
	pl := p.Bits()
	if !nat64ValidPrefixLengths[pl] {
		return "", fmt.Errorf("NAT64 prefix length must be 32, 40, 48, 56, 64, or 96; got /%d", pl)
	}
	v4 := nat64ExtractIPv4(ip.As16(), pl)
	return netip.AddrFrom4(v4).String(), nil
}

// nat64IPv6PrefixLenForIPv4 returns the IPv6 prefix length for an IPv4 /ipv4PL
// under a NAT64 prefix of length nat64PL. Only /64 and /96 are supported.
func nat64IPv6PrefixLenForIPv4(nat64PL, ipv4PL int) (int, error) {
	switch nat64PL {
	case 96:
		return 96 + ipv4PL, nil
	case 64:
		return 72 + ipv4PL, nil
	default:
		return 0, fmt.Errorf("NAT64 prefix /%d produces non-contiguous IPv4 bit ranges; "+
			"only /64 and /96 are supported for CIDR conversion", nat64PL)
	}
}

// NAT64IPv4CIDRToIPv6CIDR converts an IPv4 CIDR to its equivalent IPv6 CIDR
// under the given NAT64 prefix. Only /64 and /96 NAT64 prefixes are supported.
func NAT64SynthesizeCIDR(ipv4CIDRStr, nat64PrefixStr string, useMixed bool) (string, error) {
	ipv4Prefix, err := ParsePrefix(ipv4CIDRStr)
	if err != nil {
		return "", err
	}
	if !ipv4Prefix.Addr().Is4() {
		return "", fmt.Errorf("expected IPv4 CIDR, got %q", ipv4CIDRStr)
	}
	nat64Prefix, err := netip.ParsePrefix(nat64PrefixStr)
	if err != nil {
		return "", fmt.Errorf("invalid NAT64 prefix %q: %w", nat64PrefixStr, err)
	}
	nat64Prefix = nat64Prefix.Masked()
	if nat64Prefix.Addr().Is4() {
		return "", fmt.Errorf("NAT64 prefix must be IPv6, got %q", nat64PrefixStr)
	}
	if !nat64ValidPrefixLengths[nat64Prefix.Bits()] {
		return "", fmt.Errorf("NAT64 prefix length must be 32, 40, 48, 56, 64, or 96; got /%d", nat64Prefix.Bits())
	}
	ipv6PL, err := nat64IPv6PrefixLenForIPv4(nat64Prefix.Bits(), ipv4Prefix.Bits())
	if err != nil {
		return "", err
	}
	raw := nat64Prefix.Addr().As16()
	nat64PlaceIPv4(&raw, ipv4Prefix.Addr().As4(), nat64Prefix.Bits())
	ipv6CIDR := netip.PrefixFrom(netip.AddrFrom16(raw), ipv6PL).Masked()
	if useMixed {
		addrRaw := ipv6CIDR.Addr().As16()
		return fmt.Sprintf("%s/%d", ipToMixedNotation(addrRaw), ipv6PL), nil
	}
	return ipv6CIDR.String(), nil
}

// NAT64SynthesizeCIDRs converts a list of IPv4 CIDRs to their NAT64 IPv6 equivalents.
func NAT64SynthesizeCIDRs(ipv4CIDRs []string, nat64PrefixStr string, useMixed bool) ([]string, error) {
	result := make([]string, 0, len(ipv4CIDRs))
	for _, c := range ipv4CIDRs {
		s, err := NAT64SynthesizeCIDR(c, nat64PrefixStr, useMixed)
		if err != nil {
			return nil, fmt.Errorf("converting %q: %w", c, err)
		}
		result = append(result, s)
	}
	return result, nil
}

// ---- NPTv6 (RFC 6296) ----

func ocsSum(data []byte) uint16 {
	var acc uint32
	for i := 0; i+1 < len(data); i += 2 {
		acc += uint32(data[i])<<8 | uint32(data[i+1])
	}
	for acc > 0xFFFF {
		acc = (acc >> 16) + (acc & 0xFFFF)
	}
	return uint16(acc)
}

func ocsAdd(a, b uint16) uint16 {
	sum := uint32(a) + uint32(b)
	for sum > 0xFFFF {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}
	return uint16(sum)
}

// NPTv6Translate translates an IPv6 address from one /48 prefix to another
// using the checksum-neutral algorithm defined in RFC 6296.
func NPTv6Translate(ipv6Str, fromPrefixStr, toPrefixStr string) (string, error) {
	ip, err := ParseAddr(ipv6Str)
	if err != nil {
		return "", err
	}
	if ip.Is4() {
		return "", fmt.Errorf("expected IPv6 address, got %q", ipv6Str)
	}
	fromPrefix, err := netip.ParsePrefix(fromPrefixStr)
	if err != nil {
		return "", fmt.Errorf("invalid from-prefix %q: %w", fromPrefixStr, err)
	}
	fromPrefix = fromPrefix.Masked()
	toPrefix, err := netip.ParsePrefix(toPrefixStr)
	if err != nil {
		return "", fmt.Errorf("invalid to-prefix %q: %w", toPrefixStr, err)
	}
	toPrefix = toPrefix.Masked()
	if fromPrefix.Addr().Is4() || toPrefix.Addr().Is4() {
		return "", fmt.Errorf("NPTv6 prefixes must be IPv6")
	}
	if fromPrefix.Bits() != 48 || toPrefix.Bits() != 48 {
		return "", fmt.Errorf("NPTv6 requires /48 prefixes; got /%d and /%d",
			fromPrefix.Bits(), toPrefix.Bits())
	}
	if !fromPrefix.Contains(ip) {
		return "", fmt.Errorf("address %s is not within from-prefix %s", ipv6Str, fromPrefixStr)
	}
	result := ip.As16()
	fromRaw := fromPrefix.Addr().As16()
	toRaw := toPrefix.Addr().As16()
	copy(result[:6], toRaw[:6])
	oldOCS := ocsSum(fromRaw[:6])
	newOCS := ocsSum(toRaw[:6])
	delta := ocsAdd(newOCS, ^oldOCS)
	iidWord := uint16(result[8])<<8 | uint16(result[9])
	adjusted := ocsAdd(iidWord, ^delta)
	if adjusted == 0xFFFF {
		adjusted = 0x0000
	}
	result[8] = byte(adjusted >> 8)
	result[9] = byte(adjusted)
	return netip.AddrFrom16(result).String(), nil
}

// ---- Dual / mixed notation ----

func ipToMixedNotation(raw [16]byte) string {
	groups := [6]uint16{
		uint16(raw[0])<<8 | uint16(raw[1]),
		uint16(raw[2])<<8 | uint16(raw[3]),
		uint16(raw[4])<<8 | uint16(raw[5]),
		uint16(raw[6])<<8 | uint16(raw[7]),
		uint16(raw[8])<<8 | uint16(raw[9]),
		uint16(raw[10])<<8 | uint16(raw[11]),
	}
	v4Str := fmt.Sprintf("%d.%d.%d.%d", raw[12], raw[13], raw[14], raw[15])

	bestStart, bestLen := -1, 0
	curStart, curLen := -1, 0
	for i, g := range groups {
		if g == 0 {
			if curStart < 0 {
				curStart = i
			}
			curLen++
			if curLen > bestLen {
				bestStart, bestLen = curStart, curLen
			}
		} else {
			curStart, curLen = -1, 0
		}
	}
	if bestLen < 2 {
		bestStart = -1
	}

	var sb strings.Builder
	if bestStart < 0 {
		for i, g := range groups {
			if i > 0 {
				sb.WriteByte(':')
			}
			fmt.Fprintf(&sb, "%x", g)
		}
		sb.WriteByte(':')
	} else {
		for i := 0; i < bestStart; i++ {
			if i > 0 {
				sb.WriteByte(':')
			}
			fmt.Fprintf(&sb, "%x", groups[i])
		}
		sb.WriteString("::")
		afterStart := bestStart + bestLen
		for i := afterStart; i < 6; i++ {
			fmt.Fprintf(&sb, "%x", groups[i])
			sb.WriteByte(':')
		}
		if afterStart >= 6 && !strings.HasSuffix(sb.String(), ":") {
			sb.WriteByte(':')
		}
	}
	sb.WriteString(v4Str)
	return sb.String()
}

// IPToMixedNotation returns an IPv6 address formatted as x:x:x:x:x:x:d.d.d.d.
func IPToMixedNotation(ipStr string) (string, error) {
	ip, err := ParseAddr(ipStr)
	if err != nil {
		return "", err
	}
	if ip.Is4() {
		return ip.String(), nil
	}
	return ipToMixedNotation(ip.As16()), nil
}

// IPv4ToIPv4Mapped returns the IPv4-mapped IPv6 representation (::ffff:d.d.d.d).
func IPv4ToIPv4Mapped(ipv4Str string) (string, error) {
	ip, err := ParseAddr(ipv4Str)
	if err != nil {
		return "", err
	}
	if !ip.Is4() {
		return "", fmt.Errorf("expected IPv4 address, got %q", ipv4Str)
	}
	a4 := ip.As4()
	var raw [16]byte
	raw[10] = 0xff
	raw[11] = 0xff
	raw[12] = a4[0]
	raw[13] = a4[1]
	raw[14] = a4[2]
	raw[15] = a4[3]
	return ipToMixedNotation(raw), nil
}
