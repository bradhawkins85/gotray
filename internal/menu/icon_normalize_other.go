//go:build !windows

package menu

func platformNormalizeIcon(data []byte) []byte {
	return data
}
