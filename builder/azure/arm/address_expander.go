// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
)

const maxCNAMEHops = 10

type addressResolver interface {
	LookupCNAME(host string) (string, error)
	LookupIPAddr(host string) ([]net.IPAddr, error)
}

type netResolver struct{}

var defaultAddressResolver addressResolver = netResolver{}

func (netResolver) LookupIPAddr(host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(context.Background(), host)
}

func (netResolver) LookupCNAME(host string) (string, error) {
	return net.DefaultResolver.LookupCNAME(context.Background(), host)
}

func expandMixedAddressList(entries []string, resolver addressResolver) ([]string, error) {
	if resolver == nil {
		resolver = defaultAddressResolver
	}

	cache := map[string][]string{}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(entries))

	for _, entry := range entries {
		if net.ParseIP(entry) != nil {
			result = appendUniqueAddress(result, seen, entry)
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err == nil {
			result = appendUniqueAddress(result, seen, entry)
			continue
		}

		host := normalizeHostname(entry)
		resolved, ok := cache[host]
		if !ok {
			canonicalHost, err := resolveCanonicalHostname(host, resolver)
			if err != nil {
				return nil, err
			}

			ips, err := resolver.LookupIPAddr(canonicalHost)
			if err != nil {
				return nil, fmt.Errorf("resolve %s: %w", host, err)
			}
			resolved = make([]string, 0, len(ips))
			for _, ip := range ips {
				if ip.IP == nil {
					continue
				}
				resolved = appendUniqueString(resolved, ip.IP.String())
			}
			if len(resolved) == 0 {
				return nil, fmt.Errorf("resolve %s: no usable addresses found", host)
			}
			sort.Strings(resolved)
			cache[host] = resolved
		}

		for _, resolvedEntry := range resolved {
			result = appendUniqueAddress(result, seen, resolvedEntry)
		}
	}

	sort.Strings(result)
	return result, nil
}

func resolveCanonicalHostname(host string, resolver addressResolver) (string, error) {
	current := host
	seen := map[string]struct{}{current: {}}

	for hop := 0; hop < maxCNAMEHops; hop++ {
		next, err := resolver.LookupCNAME(current)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", host, err)
		}

		normalizedNext := normalizeHostname(next)
		if normalizedNext == "" || normalizedNext == current {
			return current, nil
		}
		if _, ok := seen[normalizedNext]; ok {
			return "", fmt.Errorf("resolve %s: CNAME loop detected", host)
		}

		seen[normalizedNext] = struct{}{}
		current = normalizedNext
	}

	return "", fmt.Errorf("resolve %s: CNAME chain exceeded %d hops", host, maxCNAMEHops)
}

func normalizeHostname(host string) string {
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

func appendUniqueAddress(dst []string, seen map[string]struct{}, value string) []string {
	if _, ok := seen[value]; ok {
		return dst
	}
	seen[value] = struct{}{}
	return append(dst, value)
}

func appendUniqueString(dst []string, value string) []string {
	for _, existing := range dst {
		if existing == value {
			return dst
		}
	}
	return append(dst, value)
}
