// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

const defaultLookupTTL = 30 * time.Second

type lookupIPAddrFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

// defaultAddressLookup is the production resolver used when no lookup is
// explicitly provided. It is not mutated by tests; tests inject a fake
// resolver via Config.addressLookup instead.
var defaultAddressLookup lookupIPAddrFunc = net.DefaultResolver.LookupIPAddr

func expandMixedAddressList(ctx context.Context, entries []string, lookup lookupIPAddrFunc) ([]string, error) {
	if lookup == nil {
		lookup = defaultAddressLookup
	}

	ctx, cancel := context.WithTimeout(ctx, defaultLookupTTL)
	defer cancel()

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
			ips, err := lookup(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("resolve %s: %w", host, err)
			}

			localSeen := map[string]struct{}{}
			resolved = make([]string, 0, len(ips))
			for _, ip := range ips {
				if ip.IP == nil {
					continue
				}
				value := ip.IP.String()
				if _, ok := localSeen[value]; ok {
					continue
				}
				localSeen[value] = struct{}{}
				resolved = append(resolved, value)
			}
			if len(resolved) == 0 {
				return nil, fmt.Errorf("resolve %s: no usable addresses found", host)
			}

			sort.Strings(resolved)
			cache[host] = resolved
		}

		for _, value := range resolved {
			result = appendUniqueAddress(result, seen, value)
		}
	}

	sort.Strings(result)
	return result, nil
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

// validateHostnamesResolve performs a pre-flight DNS check on address list
// entries during Config.Prepare so that unresolvable hostnames surface as
// warnings before the build starts, rather than as confusing
// template-construction failures at build time. Returns warning strings for
// each hostname that could not be resolved; an empty slice means all
// hostnames resolved successfully.
func validateHostnamesResolve(entries []string, lookup lookupIPAddrFunc) []string {
	if lookup == nil {
		lookup = defaultAddressLookup
	}
	var warnings []string
	for _, entry := range entries {
		if net.ParseIP(entry) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err == nil {
			continue
		}
		host := normalizeHostname(entry)
		ctx, cancel := context.WithTimeout(context.Background(), defaultLookupTTL)
		_, err := lookup(ctx, host)
		cancel()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("hostname %q could not be resolved: %v; build may fail if this hostname is still unresolvable at build time", host, err))
		}
	}
	return warnings
}
