package tokenstorage

var token = ""

func AddToken(tokenArg string) {
	token = tokenArg
}

func CheckToken(tokenArg string) bool {
	return token == tokenArg
}
