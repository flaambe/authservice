package errs

type RequestError struct {
	Status  int
	Message string
}

func (e *RequestError) Error() string {
	return e.Message
}

func New(code int, message string) *RequestError {
	return &RequestError{
		Status:  code,
		Message: message,
	}
}
