//go:build darwin

package update

import (
	"debug/macho"
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

// The Mach-O load commands that carry the oldest macOS a binary runs on.
const (
	loadCommandBuildVersion    = 0x32 // LC_BUILD_VERSION: minos at byte 12
	loadCommandVersionMinMacOS = 0x24 // LC_VERSION_MIN_MACOSX: version at byte 8
)

// checkOSMinimum refuses a binary whose Mach-O header names a newer macOS than
// the one running. A new Go toolchain raises that minimum, and the binary then
// fails on the older system after the working one is gone.
//
// It fails open: a header or a host version it cannot read leaves the decision
// to the `--version` run that follows.
func checkOSMinimum(path string) error {
	required, ok := binaryMacOSMinimum(path)
	if !ok {
		return nil
	}
	host, err := syscall.Sysctl("kern.osproductversion")
	if err != nil {
		return nil
	}
	running, ok := parseMacOSVersion(host)
	if !ok {
		return nil
	}
	if running < required {
		return fmt.Errorf("needs macOS %s or later; this Mac runs macOS %s", formatMacOSVersion(required), host)
	}
	return nil
}

// binaryMacOSMinimum returns the minimum macOS of the Mach-O file at path,
// encoded as Mach-O encodes it: major<<16 | minor<<8 | patch.
func binaryMacOSMinimum(path string) (uint32, bool) {
	file, err := macho.Open(path)
	if err != nil {
		return 0, false
	}
	defer func() { _ = file.Close() }() // read-only handle
	for _, load := range file.Loads {
		raw := load.Raw()
		if len(raw) < 16 {
			continue
		}
		switch file.ByteOrder.Uint32(raw) {
		case loadCommandBuildVersion:
			return file.ByteOrder.Uint32(raw[12:]), true
		case loadCommandVersionMinMacOS:
			return file.ByteOrder.Uint32(raw[8:]), true
		}
	}
	return 0, false
}

// parseMacOSVersion encodes "13", "13.4" or "13.4.1" the way Mach-O does.
func parseMacOSVersion(version string) (uint32, bool) {
	fields := strings.Split(strings.TrimSpace(version), ".")
	if len(fields) > 3 {
		return 0, false
	}
	var encoded uint32
	for i, shift := range []uint{16, 8, 0} {
		if i >= len(fields) {
			break
		}
		bits := 8
		if i == 0 {
			bits = 16
		}
		n, err := strconv.ParseUint(fields[i], 10, bits)
		if err != nil {
			return 0, false
		}
		encoded |= uint32(n) << shift
	}
	return encoded, true
}

func formatMacOSVersion(encoded uint32) string {
	return fmt.Sprintf("%d.%d", encoded>>16, encoded>>8&0xff)
}
