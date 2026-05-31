// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
)

type fakeAddressResolver struct {
	lookupCNAMECalls []string
	lookupIPCalls    []string
	lookupCNAME      map[string]string
	lookupCNAMEErrs  map[string]error
	lookupIPs        map[string][]net.IPAddr
	lookupIPErrs     map[string]error
}

func (f *fakeAddressResolver) LookupCNAME(host string) (string, error) {
	f.lookupCNAMECalls = append(f.lookupCNAMECalls, host)
	if err, ok := f.lookupCNAMEErrs[host]; ok {
		return "", err
	}
	if cname, ok := f.lookupCNAME[host]; ok {
		return cname, nil
	}
	return host, nil
}

func (f *fakeAddressResolver) LookupIPAddr(host string) ([]net.IPAddr, error) {
	f.lookupIPCalls = append(f.lookupIPCalls, host)
	if err, ok := f.lookupIPErrs[host]; ok {
		return nil, err
	}
	if ips, ok := f.lookupIPs[host]; ok {
		return ips, nil
	}
	return nil, nil
}

func TestExpandMixedAddressList_ResolvesSingleHostname(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}},
		},
	}

	got, err := expandMixedAddressList([]string{"ci.example.com"}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_ResolvesMultipleAddresses(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.11")}, {IP: net.ParseIP("203.0.113.10")}},
		},
	}

	got, err := expandMixedAddressList([]string{"ci.example.com"}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10", "203.0.113.11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_NormalizesEquivalentHostnames(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("203.0.113.11")}},
		},
	}

	got, err := expandMixedAddressList([]string{"ci.example.com", "CI.EXAMPLE.COM", "ci.example.com."}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10", "203.0.113.11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if !reflect.DeepEqual(resolver.lookupIPCalls, []string{"ci.example.com"}) {
		t.Fatalf("expected one normalized lookup, got %v", resolver.lookupIPCalls)
	}
}

func TestExpandMixedAddressList_KeepsLiteralIpAndCidrInputs(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("198.51.100.20")}},
		},
	}

	got, err := expandMixedAddressList([]string{"203.0.113.10/32", "198.51.100.0/24", "ci.example.com"}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"198.51.100.0/24", "198.51.100.20", "203.0.113.10/32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_FollowsCnameChain(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupCNAME: map[string]string{
			"ci.example.com":        "runner-lb.example.net.",
			"runner-lb.example.net": "runner-lb.example.net",
		},
		lookupIPs: map[string][]net.IPAddr{
			"runner-lb.example.net": {
				{IP: net.ParseIP("203.0.113.10")},
				{IP: net.ParseIP("203.0.113.11")},
			},
		},
	}

	got, err := expandMixedAddressList([]string{"ci.example.com"}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10", "203.0.113.11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if !reflect.DeepEqual(resolver.lookupCNAMECalls, []string{"ci.example.com", "runner-lb.example.net"}) {
		t.Fatalf("expected explicit CNAME traversal, got %v", resolver.lookupCNAMECalls)
	}
	if !reflect.DeepEqual(resolver.lookupIPCalls, []string{"runner-lb.example.net"}) {
		t.Fatalf("expected final IP lookup on canonical host, got %v", resolver.lookupIPCalls)
	}
}

func TestExpandMixedAddressList_DeduplicatesAddresses(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci-a.example.com": {{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("203.0.113.10")}},
			"ci-b.example.com": {{IP: net.ParseIP("203.0.113.10")}},
		},
	}

	got, err := expandMixedAddressList([]string{"ci-a.example.com", "ci-b.example.com"}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_FailsOnEmptyAnswerSet(t *testing.T) {
	resolver := &fakeAddressResolver{lookupIPs: map[string][]net.IPAddr{"ci.example.com": {}}}

	_, err := expandMixedAddressList([]string{"ci.example.com"}, resolver)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !containsAll(got, "ci.example.com") {
		t.Fatalf("expected hostname in error, got %q", got)
	}
}

func TestExpandMixedAddressList_FailsOnLookupError(t *testing.T) {
	resolver := &fakeAddressResolver{lookupIPErrs: map[string]error{"ci.example.com": errors.New("lookup failed")}}

	_, err := expandMixedAddressList([]string{"ci.example.com"}, resolver)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !containsAll(got, "ci.example.com", "lookup failed") {
		t.Fatalf("expected hostname and resolver error, got %q", got)
	}
}

func TestExpandMixedAddressList_FailsWholeInputOnMixedGoodAndBadEntries(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}},
		},
		lookupIPErrs: map[string]error{
			"proxy.example.com": errors.New("nxdomain"),
		},
	}

	got, err := expandMixedAddressList([]string{"203.0.113.10/32", "ci.example.com", "proxy.example.com"}, resolver)
	if err == nil {
		t.Fatal("expected error")
	}
	if got != nil {
		t.Fatalf("expected no partial allowlist, got %v", got)
	}
	if msg := err.Error(); !containsAll(msg, "proxy.example.com", "nxdomain") {
		t.Fatalf("expected hostname and resolver error, got %q", msg)
	}
}

func TestExpandMixedAddressList_HandlesIpv6AccordingToPolicy(t *testing.T) {
	resolver := &fakeAddressResolver{
		lookupIPs: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("2001:db8::10")}},
		},
	}

	got, err := expandMixedAddressList([]string{"ci.example.com"}, resolver)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"2001:db8::10", "203.0.113.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}
