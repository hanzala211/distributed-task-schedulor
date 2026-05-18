package service

import (
	"fmt"
)

type AppError struct {
	Message string
	Err     error
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}
