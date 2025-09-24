package templates

type API struct {
	OperationID string
	Method      string
	Path        string
	Description string
	RequestBody *RequestBody
	Response    *Response
}

type RequestBody struct {
	ModelName string
}

type Response struct {
	ModelName string
	Status    string
}

type APIs map[string][]API
