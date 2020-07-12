package utils

func ListContainsString(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func ListRemoveString(list []string, value string) (result []string) {
	for _, v := range list {
		if v == value {
			continue
		}
		result = append(result, v)
	}
	return result
}
