//go:build windows
// +build windows

package trmm

import (
	"strconv"

	"golang.org/x/sys/windows/registry"
)

func readRegistrySettings() map[string]string {
	result := make(map[string]string)

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\TacticalRMM`, registry.QUERY_VALUE)
	if err != nil {
		return result
	}
	defer key.Close()

	names := []string{"BaseURL", "AgentID", "AgentPK", "AgentPk", "SiteID", "ClientID"}
	for _, name := range names {
		if value, _, err := key.GetStringValue(name); err == nil {
			result[name] = value
			continue
		}
		if value, _, err := key.GetIntegerValue(name); err == nil {
			result[name] = strconv.FormatUint(value, 10)
		}
	}
	return result
}
