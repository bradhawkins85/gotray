package menu

func cloneDefaultIcon() []byte {
	cp := make([]byte, len(defaultIconData))
	copy(cp, defaultIconData)
	return cp
}

func cloneIcon(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp
}

func normalizedIcon(data []byte) []byte {
	if len(data) == 0 {
		return cloneDefaultIcon()
	}
	normalized := platformNormalizeIcon(data)
	if len(normalized) == 0 {
		return cloneDefaultIcon()
	}
	return cloneIcon(normalized)
}
