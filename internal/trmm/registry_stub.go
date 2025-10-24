//go:build !windows
// +build !windows

package trmm

func readRegistrySettings() map[string]string {
	return map[string]string{}
}
