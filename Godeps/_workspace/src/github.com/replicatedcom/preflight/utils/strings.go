package utils

func StringInSlice(a string, s []string) bool {
	for _, b := range s {
		if b == a {
			return true
		}
	}
	return false
}
