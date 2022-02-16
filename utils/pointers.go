package utils

func StringOrEmpty(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

func Int32OrDefault(p *int32, d int32) int32 {
	if p == nil {
		return d
	}

	return *p
}
