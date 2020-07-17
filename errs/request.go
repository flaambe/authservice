package errs

type RequestError struct {
	Status  int
	Message string
	Err     error
}

func (e *RequestError) Error() string {
	return e.Message
}

func New(status int, message string, err error) *RequestError {
	return &RequestError{
		Status:  status,
		Message: message,
		Err:     err,
	}
}
