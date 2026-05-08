package network

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the network/IP-utility provider functions registered by
// terraform-burnham (CIDR set ops, queries, NAT64, NPTv6, IP arithmetic, IPAM).
func Functions() []func() function.Function {
	return []func() function.Function{
		// CIDR set operations
		NewCIDRMergeFunction,
		NewCIDRSubtractFunction,
		NewCIDRIntersectFunction,
		NewCIDRExpandFunction,
		NewCIDREnumerateFunction,
		NewRangeToCIDRsFunction,
		// Query / containment
		NewIPInCIDRFunction,
		NewCIDRsContainingIPFunction,
		NewCIDRContainsFunction,
		NewCIDROverlapsFunction,
		NewCIDRsOverlapAnyFunction,
		NewCIDRsAreDisjointFunction,
		// CIDR information
		NewCIDRHostCountFunction,
		NewCIDRUsableHostCountFunction,
		NewCIDRFirstIPFunction,
		NewCIDRLastIPFunction,
		NewCIDRPrefixLengthFunction,
		NewCIDRWildcardFunction,
		// Version detection / filtering
		NewIPVersionFunction,
		NewCIDRVersionFunction,
		NewCIDRFilterVersionFunction,
		// Private-range checks
		NewCIDRIsPrivateFunction,
		NewIPIsPrivateFunction,
		// IP arithmetic
		NewIPAddFunction,
		NewIPSubtractFunction,
		// NAT64 (RFC 6052)
		NewNAT64SynthesizeFunction,
		NewNAT64ExtractFunction,
		NewNAT64PrefixValidFunction,
		NewNAT64SynthesizeCIDRFunction,
		NewNAT64SynthesizeCIDRsFunction,
		// NPTv6 (RFC 6296)
		NewNPTv6TranslateFunction,
		// Dual / mixed notation
		NewIPToMixedNotationFunction,
		NewIPv4ToIPv4MappedFunction,
		// IPAM
		NewCIDRFindFreeFunction,
		// IP over Avian Carriers (RFC 1149 / RFC 2549)
		NewPigeonThroughputFunction,
	}
}
