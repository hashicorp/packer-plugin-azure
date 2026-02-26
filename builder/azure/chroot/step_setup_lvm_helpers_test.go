// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build linux || freebsd

package chroot

import (
	"testing"
)

func Test_isMountableLV(t *testing.T) {
	tests := []struct {
		name string
		attr string
		want bool
	}{
		{"empty attr is mountable", "", true},
		{"normal volume -wi-a-----", "-wi-a-----", true},
		{"origin volume owi-a-s---", "owi-a-s---", true},
		{"snapshot", "swi-a-s---", false},
		{"Snapshot uppercase", "Swi-a-s---", false},
		{"virtual", "vwi-a-t---", false},
		{"Virtual uppercase", "Vwi-a-t---", false},
		{"thin pool", "twi-a-t---", false},
		{"Thin pool uppercase", "Twi-a-t---", false},
		{"RAID metadata", "ewi-a-----", false},
		{"RAID metadata uppercase", "Ewi-a-----", false},
		{"internal", "iwi-------", false},
		{"Internal uppercase", "Iwi-------", false},
		{"mirror log", "lwi-a-----", false},
		{"Mirror log uppercase", "Lwi-a-----", false},
		{"mirror image", "dwi-a-----", false},
		{"Mirror image uppercase", "Dwi-a-----", false},
		{"pvmove", "pwi-------", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMountableLV(tt.attr)
			if got != tt.want {
				t.Errorf("isMountableLV(%q) = %v, want %v", tt.attr, got, tt.want)
			}
		})
	}
}

func Test_lvTypeDescription(t *testing.T) {
	tests := []struct {
		attr string
		want string
	}{
		{"", "unknown type"},
		{"swi-a-s---", "snapshot"},
		{"Swi-a-s---", "snapshot"},
		{"vwi-a-t---", "virtual volume"},
		{"twi-a-t---", "thin pool"},
		{"ewi-a-----", "RAID/pool metadata"},
		{"iwi-------", "internal volume"},
		{"lwi-a-----", "mirror log"},
		{"dwi-a-----", "mirror/RAID image"},
		{"pwi-------", "pvmove volume"},
		{"-wi-a-----", "unknown type"},
		{"owi-a-s---", "unknown type"},
	}
	for _, tt := range tests {
		t.Run(tt.attr, func(t *testing.T) {
			got := lvTypeDescription(tt.attr)
			if got != tt.want {
				t.Errorf("lvTypeDescription(%q) = %q, want %q", tt.attr, got, tt.want)
			}
		})
	}
}

func Test_resolveMapperHeuristic(t *testing.T) {
	tests := []struct {
		name     string
		basename string
		want     string
	}{
		{
			name:     "simple VG-LV",
			basename: "rhel-root",
			want:     "rhel/root",
		},
		{
			name:     "VG with escaped dash",
			basename: "my--vg-root",
			want:     "my-vg/root",
		},
		{
			name:     "LV with escaped dash",
			basename: "rhel-my--lv",
			want:     "rhel/my-lv",
		},
		{
			name:     "both escaped",
			basename: "my--vg-my--lv",
			want:     "my-vg/my-lv",
		},
		{
			name:     "multiple dashes in VG",
			basename: "a--b--c-root",
			want:     "a-b-c/root",
		},
		{
			name:     "no dash at all",
			basename: "nodash",
			want:     "",
		},
		{
			name:     "only double dashes (no separator)",
			basename: "all--dashes",
			want:     "",
		},
		{
			name:     "multiple single-dash separators picks last",
			basename: "vg-lv-extra",
			want:     "vg-lv/extra",
		},
		{
			name:     "centos style",
			basename: "centos-root",
			want:     "centos/root",
		},
		{
			name:     "fedora style",
			basename: "fedora_server-root",
			want:     "fedora_server/root",
		},
		{
			name:     "single char VG and LV",
			basename: "a-b",
			want:     "a/b",
		},
		{
			name:     "empty string",
			basename: "",
			want:     "",
		},
		{
			name:     "single char",
			basename: "a",
			want:     "",
		},
		{
			name:     "dash at start",
			basename: "-root",
			want:     "",
		},
		{
			name:     "dash at end",
			basename: "vg-",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveMapperHeuristic(tt.basename)
			if got != tt.want {
				t.Errorf("resolveMapperHeuristic(%q) = %q, want %q", tt.basename, got, tt.want)
			}
		})
	}
}

func Test_resolveDevPath(t *testing.T) {
	tests := []struct {
		name   string
		device string
		want   string
	}{
		{
			name:   "standard /dev/vg/lv",
			device: "/dev/rhel/root",
			want:   "rhel/root",
		},
		{
			name:   "longer path /dev/volgroup/logvol",
			device: "/dev/mygroup/myvolume",
			want:   "mygroup/myvolume",
		},
		{
			name:   "too short /dev/foo",
			device: "/dev/foo",
			want:   "",
		},
		{
			name:   "root path /",
			device: "/",
			want:   "",
		},
		{
			name:   "empty",
			device: "",
			want:   "",
		},
		{
			name:   "deep path keeps last two segments",
			device: "/dev/some/deep/path/vg/lv",
			want:   "vg/lv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDevPath(tt.device)
			if got != tt.want {
				t.Errorf("resolveDevPath(%q) = %q, want %q", tt.device, got, tt.want)
			}
		})
	}
}

func Test_resolveVGLV(t *testing.T) {
	// resolveVGLV tries dmsetup first (will fail on non-Linux/no-LVM systems),
	// then falls back to the heuristic. We test the full function to exercise
	// both code paths; on systems without dmsetup the heuristic is used.
	tests := []struct {
		name   string
		device string
		want   string
	}{
		{
			name:   "mapper path simple",
			device: "/dev/mapper/rhel-root",
			want:   "rhel/root",
		},
		{
			name:   "mapper path with escaped dashes",
			device: "/dev/mapper/my--vg-root",
			want:   "my-vg/root",
		},
		{
			name:   "dev vg lv path",
			device: "/dev/rhel/root",
			want:   "rhel/root",
		},
		{
			name:   "unrecognized path",
			device: "/dev/sda1",
			want:   "",
		},
		{
			name:   "empty",
			device: "",
			want:   "",
		},
		{
			name:   "mapper no dash",
			device: "/dev/mapper/nodash",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVGLV(tt.device)
			if got != tt.want {
				t.Errorf("resolveVGLV(%q) = %q, want %q", tt.device, got, tt.want)
			}
		})
	}
}

func Test_vgFromDevicePath(t *testing.T) {
	tests := []struct {
		name   string
		device string
		want   string
	}{
		{
			name:   "mapper path",
			device: "/dev/mapper/rhel-root",
			want:   "rhel",
		},
		{
			name:   "mapper path escaped dashes",
			device: "/dev/mapper/my--vg-root",
			want:   "my-vg",
		},
		{
			name:   "dev path",
			device: "/dev/rhel/root",
			want:   "rhel",
		},
		{
			name:   "unrecognized",
			device: "/dev/sda1",
			want:   "",
		},
		{
			name:   "empty",
			device: "",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vgFromDevicePath(tt.device)
			if got != tt.want {
				t.Errorf("vgFromDevicePath(%q) = %q, want %q", tt.device, got, tt.want)
			}
		})
	}
}

func Test_selectRootLV(t *testing.T) {
	tests := []struct {
		name       string
		candidates []lvInfo
		want       string
	}{
		{
			name:       "empty list returns empty",
			candidates: nil,
			want:       "",
		},
		{
			name: "single candidate returns it",
			candidates: []lvInfo{
				{name: "data", vg: "rhel", path: "/dev/rhel/data", attr: "-wi-a-----"},
			},
			want: "/dev/rhel/data",
		},
		{
			name: "exact match root",
			candidates: []lvInfo{
				{name: "swap", vg: "rhel", path: "/dev/rhel/swap", attr: "-wi-a-----"},
				{name: "root", vg: "rhel", path: "/dev/rhel/root", attr: "-wi-a-----"},
				{name: "home", vg: "rhel", path: "/dev/rhel/home", attr: "-wi-a-----"},
			},
			want: "/dev/rhel/root",
		},
		{
			name: "exact match lv_root",
			candidates: []lvInfo{
				{name: "home", vg: "centos", path: "/dev/centos/home", attr: "-wi-a-----"},
				{name: "lv_root", vg: "centos", path: "/dev/centos/lv_root", attr: "-wi-a-----"},
			},
			want: "/dev/centos/lv_root",
		},
		{
			name: "exact match rootlv",
			candidates: []lvInfo{
				{name: "data", vg: "vg0", path: "/dev/vg0/data", attr: "-wi-a-----"},
				{name: "rootlv", vg: "vg0", path: "/dev/vg0/rootlv", attr: "-wi-a-----"},
			},
			want: "/dev/vg0/rootlv",
		},
		{
			name: "exact match lvroot",
			candidates: []lvInfo{
				{name: "data", vg: "vg0", path: "/dev/vg0/data", attr: "-wi-a-----"},
				{name: "lvroot", vg: "vg0", path: "/dev/vg0/lvroot", attr: "-wi-a-----"},
			},
			want: "/dev/vg0/lvroot",
		},
		{
			name: "exact match is case-insensitive",
			candidates: []lvInfo{
				{name: "data", vg: "rhel", path: "/dev/rhel/data", attr: "-wi-a-----"},
				{name: "Root", vg: "rhel", path: "/dev/rhel/Root", attr: "-wi-a-----"},
			},
			want: "/dev/rhel/Root",
		},
		{
			name: "partial match containing root",
			candidates: []lvInfo{
				{name: "data", vg: "vg0", path: "/dev/vg0/data", attr: "-wi-a-----"},
				{name: "my_rootfs", vg: "vg0", path: "/dev/vg0/my_rootfs", attr: "-wi-a-----"},
				{name: "home", vg: "vg0", path: "/dev/vg0/home", attr: "-wi-a-----"},
			},
			want: "/dev/vg0/my_rootfs",
		},
		{
			name: "exact match takes priority over partial",
			candidates: []lvInfo{
				{name: "my_rootfs", vg: "vg0", path: "/dev/vg0/my_rootfs", attr: "-wi-a-----"},
				{name: "root", vg: "vg0", path: "/dev/vg0/root", attr: "-wi-a-----"},
			},
			want: "/dev/vg0/root",
		},
		{
			name: "no root match returns first candidate",
			candidates: []lvInfo{
				{name: "data", vg: "vg0", path: "/dev/vg0/data", attr: "-wi-a-----"},
				{name: "home", vg: "vg0", path: "/dev/vg0/home", attr: "-wi-a-----"},
				{name: "var", vg: "vg0", path: "/dev/vg0/var", attr: "-wi-a-----"},
			},
			want: "/dev/vg0/data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectRootLV(tt.candidates)
			if got != tt.want {
				t.Errorf("selectRootLV() = %q, want %q", got, tt.want)
			}
		})
	}
}
