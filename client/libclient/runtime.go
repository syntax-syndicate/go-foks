// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import "runtime"

type RuntimeGroup uint

const (
	RuntimeGroupUnknown RuntimeGroup = 0
	RuntimeGroupLinux   RuntimeGroup = 1
	RuntimeGroupDarwin  RuntimeGroup = 2
	RuntimeGroupWindows RuntimeGroup = 3
)

func GetRuntimeGroup() RuntimeGroup {
	return getRuntimeGroup(runtime.GOOS)
}

func getRuntimeGroup(osname string) RuntimeGroup {
	switch osname {
	case "linux", "dragonfly", "freebsd", "netbsd", "openbsd", "android":
		return RuntimeGroupLinux
	case "darwin", "ios":
		return RuntimeGroupDarwin
	case "windows":
		return RuntimeGroupWindows
	default:
		return RuntimeGroupUnknown
	}
}
