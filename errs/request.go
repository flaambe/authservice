package errs

type RequestError struct {
	Status  int
	Message string
}

func (e *RequestError) Error() string {
	return e.Message
}

func New(status int, message string) *RequestError {
	return &RequestError{
		Status:  status,
		Message: message,
	}
}
