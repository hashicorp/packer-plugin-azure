// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package arm

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
)

func fakeLookup(ips map[string][]net.IPAddr, errs map[string]error) lookupIPAddrFunc {
	return func(_ context.Context, host string) ([]net.IPAddr, error) {
		if err, ok := errs[host]; ok {
			return nil, err
		}
		return ips[host], nil
	}
}

type trackingLookup struct {
	ips   map[string][]net.IPAddr
	errs  map[string]error
	calls []string
}

func (t *trackingLookup) lookup(_ context.Context, host string) ([]net.IPAddr, error) {
	t.calls = append(t.calls, host)
	if err, ok := t.errs[host]; ok {
		return nil, err
	}
	return t.ips[host], nil
}

func TestExpandMixedAddressList_ResolvesSingleHostname(t *testing.T) {
	lookup := fakeLookup(
		map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}},
		},
		nil,
	)

	got, err := expandMixedAddressList([]string{"ci.example.com"}, lookup)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_ResolvesMultipleAddresses(t *testing.T) {
	lookup := fakeLookup(
		map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.11")}, {IP: net.ParseIP("203.0.113.10")}},
		},
		nil,
	)

	got, err := expandMixedAddressList([]string{"ci.example.com"}, lookup)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10", "203.0.113.11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_NormalizesEquivalentHostnames(t *testing.T) {
	tracker := &trackingLookup{
		ips: map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("203.0.113.11")}},
		},
	}

	got, err := expandMixedAddressList([]string{"ci.example.com", "CI.EXAMPLE.COM", "ci.example.com."}, tracker.lookup)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10", "203.0.113.11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if !reflect.DeepEqual(tracker.calls, []string{"ci.example.com"}) {
		t.Fatalf("expected one normalized lookup, got %v", tracker.calls)
	}
}

func TestExpandMixedAddressList_KeepsLiteralIpAndCidrInputs(t *testing.T) {
	lookup := fakeLookup(
		map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("198.51.100.20")}},
		},
		nil,
	)

	got, err := expandMixedAddressList([]string{"203.0.113.10/32", "198.51.100.0/24", "ci.example.com"}, lookup)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"198.51.100.0/24", "198.51.100.20", "203.0.113.10/32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_DeduplicatesAddresses(t *testing.T) {
	lookup := fakeLookup(
		map[string][]net.IPAddr{
			"ci-a.example.com": {{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("203.0.113.10")}},
			"ci-b.example.com": {{IP: net.ParseIP("203.0.113.10")}},
		},
		nil,
	)

	got, err := expandMixedAddressList([]string{"ci-a.example.com", "ci-b.example.com"}, lookup)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"203.0.113.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExpandMixedAddressList_FailsOnEmptyAnswerSet(t *testing.T) {
	lookup := fakeLookup(
		map[string][]net.IPAddr{"ci.example.com": {}},
		nil,
	)

	_, err := expandMixedAddressList([]string{"ci.example.com"}, lookup)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !containsAll(got, "ci.example.com") {
		t.Fatalf("expected hostname in error, got %q", got)
	}
}

func TestExpandMixedAddressList_FailsOnLookupError(t *testing.T) {
	lookup := fakeLookup(
		nil,
		map[string]error{"ci.example.com": errors.New("lookup failed")},
	)

	_, err := expandMixedAddressList([]string{"ci.example.com"}, lookup)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !containsAll(got, "ci.example.com", "lookup failed") {
		t.Fatalf("expected hostname and resolver error, got %q", got)
	}
}

func TestExpandMixedAddressList_PropagatesContextDeadline(t *testing.T) {
	lookup := fakeLookup(
		nil,
		map[string]error{"ci.example.com": context.DeadlineExceeded},
	)

	_, err := expandMixedAddressList([]string{"ci.example.com"}, lookup)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded in error chain, got %v", err)
	}
}

func TestExpandMixedAddressList_FailsWholeInputOnMixedGoodAndBadEntries(t *testing.T) {
	lookup := fakeLookup(
		map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}},
		},
		map[string]error{
			"proxy.example.com": errors.New("nxdomain"),
		},
	)

	got, err := expandMixedAddressList([]string{"203.0.113.10/32", "ci.example.com", "proxy.example.com"}, lookup)
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
	lookup := fakeLookup(
		map[string][]net.IPAddr{
			"ci.example.com": {{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("2001:db8::10")}},
		},
		nil,
	)

	got, err := expandMixedAddressList([]string{"ci.example.com"}, lookup)
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
