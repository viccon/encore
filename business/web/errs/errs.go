// Package errs provides types and support related to web error functionality.
package errs

import (
	"fmt"
	"net/http"

	"encore.dev/beta/errs"
	"encore.dev/middleware"
	"github.com/ardanlabs/encore/foundation/validate"
)

var errMap = map[int]errs.ErrCode{
	http.StatusOK:                  errs.OK,
	http.StatusInternalServerError: errs.Internal,
	http.StatusBadRequest:          errs.FailedPrecondition,
	http.StatusGatewayTimeout:      errs.DeadlineExceeded,
	http.StatusNotFound:            errs.NotFound,
	http.StatusConflict:            errs.Aborted,
	http.StatusForbidden:           errs.PermissionDenied,
	http.StatusTooManyRequests:     errs.ResourceExhausted,
	http.StatusNotImplemented:      errs.Unimplemented,
	http.StatusServiceUnavailable:  errs.Unavailable,
	http.StatusUnauthorized:        errs.Unauthenticated,
}

type extraDetails struct {
	HTTPStatusCode int                  `json:"httpStatusCode"`
	HTTPStatus     string               `json:"httpStatus"`
	Fields         validate.FieldErrors `json:"fields,omitempty"`
}

func (extraDetails) ErrDetails() {}

// New constructs an encore error based on an app error.
func New(httpStatus int, err error) *errs.Error {
	return &errs.Error{
		Code:    errMap[httpStatus],
		Message: err.Error(),
		Details: extraDetails{
			HTTPStatusCode: httpStatus,
			HTTPStatus:     http.StatusText(httpStatus),
		},
	}
}

// Newf constructs an encore error based on a error message.
func Newf(httpStatus int, format string, v ...any) *errs.Error {
	return &errs.Error{
		Code:    errMap[httpStatus],
		Message: fmt.Sprintf(format, v...),
		Details: extraDetails{
			HTTPStatusCode: httpStatus,
			HTTPStatus:     http.StatusText(httpStatus),
		},
	}
}

// NewResponse constructs an encore middleware response with a Go error.
func NewResponse(httpStatus int, err error) middleware.Response {
	return middleware.Response{
		HTTPStatus: httpStatus,
		Err: &errs.Error{
			Code:    errMap[httpStatus],
			Message: err.Error(),
			Details: extraDetails{
				HTTPStatusCode: httpStatus,
				HTTPStatus:     http.StatusText(httpStatus),
			},
		},
	}
}

// NewResponsef constructs an encore middleware response with a message.
func NewResponsef(httpStatus int, format string, v ...any) middleware.Response {
	return middleware.Response{
		HTTPStatus: httpStatus,
		Err: &errs.Error{
			Code:    errMap[httpStatus],
			Message: fmt.Sprintf(format, v...),
			Details: extraDetails{
				HTTPStatusCode: httpStatus,
				HTTPStatus:     http.StatusText(httpStatus),
			},
		},
	}
}
