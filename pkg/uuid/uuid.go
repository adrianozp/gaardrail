package uuid

import (
	"github.com/google/uuid"
)

var Uuid func() string

func New() string {
	if Uuid != nil {
		return Uuid()
	}
	return uuid.NewString()
}

func With(f func() string) {
	Uuid = f
}

func WithString(u string) {
	Uuid = func() string { return u }
}
