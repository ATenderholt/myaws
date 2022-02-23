package types

type ApiRequest struct {
	ID            int64
	Service       string
	Authorization string
	ContentType   string
	Payload       string
}
