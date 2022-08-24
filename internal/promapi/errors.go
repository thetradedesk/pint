package promapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prymitive/current"
)

func IsUnavailableError(err error) bool {
	var e1 APIError
	if ok := errors.As(err, &e1); ok {
		return e1.ErrorType == v1.ErrServer
	}

	return true
}

type APIError struct {
	Status    string       `json:"status"`
	ErrorType v1.ErrorType `json:"errorType"`
	Err       string       `json:"error"`
}

func (e APIError) Error() string {
	return e.Err
}

const (
	ErrUnknown    v1.ErrorType = "unknown"
	ErrJSONStream v1.ErrorType = "json_stream"
)

func decodeErrorType(s string) v1.ErrorType {
	switch s {
	case string(v1.ErrBadData):
		return v1.ErrBadData
	case string(v1.ErrTimeout):
		return v1.ErrTimeout
	case string(v1.ErrCanceled):
		return v1.ErrCanceled
	case string(v1.ErrExec):
		return v1.ErrExec
	case string(v1.ErrBadResponse):
		return v1.ErrBadResponse
	case string(v1.ErrServer):
		return v1.ErrServer
	case string(v1.ErrClient):
		return v1.ErrClient
	default:
		return ErrUnknown
	}
}

func decodeError(err error) string {
	if errors.Is(err, context.Canceled) {
		return context.Canceled.Error()
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return "connection refused"
	}

	var neterr net.Error
	if ok := errors.As(err, &neterr); ok && neterr.Timeout() {
		return "connection timeout"
	}

	var e1 APIError
	if ok := errors.As(err, &e1); ok {
		return fmt.Sprintf("%s: %s", e1.ErrorType, e1.Err)
	}

	return err.Error()
}

func tryDecodingAPIError(resp *http.Response) error {
	var status, errType, errText string
	decoder := current.Object(
		func() {},
		current.Key("status", current.Text(func(s string) {
			status = s
		})),
		current.Key("error", current.Text(func(s string) {
			errText = s
		})),
		current.Key("errorType", current.Text(func(s string) {
			errType = s
		})),
	)

	dec := json.NewDecoder(resp.Body)
	if err := current.Stream(dec, decoder); err != nil {
		switch resp.StatusCode / 100 {
		case 4:
			return APIError{Status: "error", ErrorType: v1.ErrClient, Err: fmt.Sprintf("client error: %d", resp.StatusCode)}
		case 5:
			return APIError{Status: "error", ErrorType: v1.ErrServer, Err: fmt.Sprintf("server error: %d", resp.StatusCode)}
		}
		return APIError{Status: "error", ErrorType: v1.ErrBadResponse, Err: fmt.Sprintf("bad response code: %d", resp.StatusCode)}
	}

	return APIError{Status: status, ErrorType: decodeErrorType(errType), Err: errText}
}
