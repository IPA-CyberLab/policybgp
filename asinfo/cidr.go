package asinfo

import (
	"fmt"
	"net/netip"
)

// ipRangeToCIDRs converts an IP address range, specified by a start and an end (inclusive),
// into the smallest possible list of CIDR prefixes that exactly cover that range.
//
// Behavior:
//   - Returns an error if start or end are invalid, if they are not of the
//     same IP family (both IPv4 or both IPv6), or if start is greater than end.
//   - The returned CIDR prefixes are ordered and cover the range without gaps or overlaps.
//   - If start equals end, a single host prefix (e.g., /32 for IPv4 or /128 for IPv6) is returned.
func ipRangeToCIDRs(start, end netip.Addr) ([]netip.Prefix, error) {
	// Validate inputs
	if !start.IsValid() {
		return nil, fmt.Errorf("start (%s) is not valid", start.String())
	}
	if !end.IsValid() {
		return nil, fmt.Errorf("end (%s) is not valid", end.String())
	}

	if start.Is4() != end.Is4() {
		return nil, fmt.Errorf("start (%s) and end (%s) must be of the same IP family", start, end)
	}

	if start.Compare(end) > 0 {
		return nil, fmt.Errorf("start (%s) must not be greater than end (%s)", start, end)
	}

	var prefixes []netip.Prefix

	curr := start
	for {
		prefix, lastAddr := findLargestPrefix(curr, end)
		prefixes = append(prefixes, prefix)

		if lastAddr.Compare(end) == 0 {
			break
		}
		curr = lastAddr.Next()
	}

	return prefixes, nil
}

// findLargestPrefix finds the largest CIDR prefix that starts at start and doesn't exceed end
// It returns the prefix and the last address in that prefix range.
func findLargestPrefix(start, end netip.Addr) (netip.Prefix, netip.Addr) {
	maxBits := start.BitLen()
	lastFit := netip.PrefixFrom(start, maxBits)
	lastFitRangeEnd := start
	for prefixLen := maxBits - 1; prefixLen >= 0; prefixLen-- {
		candidate := netip.PrefixFrom(start, prefixLen).Masked()

		if candidate.Addr().Compare(start) != 0 {
			return lastFit, lastFitRangeEnd
		}

		// Check if this prefix doesn't exceed end
		lastAddr := getLastAddr(candidate)
		if lastAddr.Compare(end) > 0 {
			return lastFit, lastFitRangeEnd
		}

		lastFit = candidate
		lastFitRangeEnd = lastAddr
	}

	// NOT REACHED
	return lastFit, lastFitRangeEnd
}

// getLastAddr returns the last address in a prefix
func getLastAddr(prefix netip.Prefix) netip.Addr {
	addr := prefix.Addr()
	prefixLen := prefix.Bits()
	maxBits := addr.BitLen()

	if prefixLen == maxBits {
		return addr // Host route
	}

	if addr.Is4() {
		ip4 := addr.As4()
		hostBits := 32 - prefixLen

		ipInt := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])

		mask := uint32((1 << hostBits) - 1)
		lastInt := ipInt + mask

		return netip.AddrFrom4([4]byte{
			byte(lastInt >> 24),
			byte(lastInt >> 16),
			byte(lastInt >> 8),
			byte(lastInt),
		})
	} else {
		ip16 := addr.As16()
		hostBits := 128 - prefixLen

		result := ip16

		bitsRemaining := hostBits
		for i := 15; i >= 0 && bitsRemaining > 0; i-- {
			if bitsRemaining >= 8 {
				result[i] = 0xFF
				bitsRemaining -= 8
			} else {
				mask := byte((1 << bitsRemaining) - 1)
				result[i] |= mask
				bitsRemaining = 0
			}
		}

		return netip.AddrFrom16(result)
	}
}
