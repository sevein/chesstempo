package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ResponseError struct {
	Code   int    `json:"-"`
	Reason string `json:"reason"`
}

func (err ResponseError) Error() string {
	if err.Reason == "" {
		err.Reason = "Unknown error."
	}
	return err.Reason
}

type appHandler func(w http.ResponseWriter, r *http.Request) error

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)

	if err == nil {
		return // Happy path!
	}

	var er *ResponseError
	if !errors.As(err, &er) {
		http.Error(w, "", http.StatusBadGateway)
		return
	}

	if ret := http.StatusText(er.Code); ret == "" {
		er.Code = http.StatusBadGateway
	}

	w.WriteHeader(er.Code)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(er); err != nil {
		http.Error(w, "", http.StatusBadGateway)
		return
	}
}
