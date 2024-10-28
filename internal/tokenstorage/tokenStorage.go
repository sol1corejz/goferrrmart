package tokenstorage

var tokens = make([]string, 0)

func contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func AddToken(tokenArg string) {
	tokens = append(tokens, tokenArg)
}

func CheckToken(tokenArg string) bool {
	return contains(tokens, tokenArg)
}
