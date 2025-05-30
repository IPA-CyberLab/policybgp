package common

type ErrAbort struct{}

func (ErrAbort) Error() string {
	return "Session aborted"
}

func (ErrAbort) ExitCode() int {
	return 130
}

type ErrInvalidInput struct {
	Msg string
}

func (e ErrInvalidInput) Error() string {
	return e.Msg
}

func (ErrInvalidInput) ExitCode() int {
	return 1
}

func (ErrInvalidInput) Is(err error) bool {
	_, ok := err.(ErrInvalidInput)
	return ok
}
