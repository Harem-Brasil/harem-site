package domain

import (
	"errors"
	"net/http"
)

// AppError transporta status HTTP e detalhes para a camada HTTP sem acoplar a Gin.
type AppError struct {
	HTTPStatus  int
	Title       string
	Detail      string
	FieldErrors map[string]string
}

func (e *AppError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Title
}

func Err(httpStatus int, detail string) *AppError {
	title := http.StatusText(httpStatus)
	if title == "" {
		title = "Error"
	}
	return &AppError{HTTPStatus: httpStatus, Title: title, Detail: detail}
}

func ErrWithTitle(httpStatus int, title, detail string) *AppError {
	return &AppError{HTTPStatus: httpStatus, Title: title, Detail: detail}
}

func ErrValidation(detail string, fields map[string]string) *AppError {
	return &AppError{
		HTTPStatus:  http.StatusUnprocessableEntity,
		Title:       "Validation Error",
		Detail:      detail,
		FieldErrors: fields,
	}
}

func AsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
