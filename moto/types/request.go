package types

type ApiRequest struct {
	ID            int64
	Service       string
	Method        string
	Path          string
	Authorization string
	ContentType   string
	Payload       string
}
