package api

const (
	// 1xx
	InternalServerError = 101
	BadArgument         = 102
	Unauthorized        = 103
	BadAuthorization    = 104

	// 2xx
	Banned                              = 201
	AccountFound                        = 202
	AccountNotFoundOrPasswordMismatched = 203
)
