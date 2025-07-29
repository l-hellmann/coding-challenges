package apiError

import (
	"strings"
)

type Error struct {
	code     int
	messages []string
}

func New(code int, message string, additional ...string) error {
	return Error{
		code:     code,
		messages: append([]string{message}, additional...),
	}
}

func (e Error) Error() string {
	return strings.Join(e.messages, "\n")
}
func (e Error) Code() int {
	return e.code
}
func (e Error) Messages() []string {
	return e.messages
}
