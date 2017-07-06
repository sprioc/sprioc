package handler

import (
	"net/http"
	"time"
)

type Middleware struct {
	*State
	M func(state *State, next http.Handler) http.Handler
}

func (m Middleware) Handler(next http.Handler) http.Handler {
	return m.M(m.State, next)
}

func Timeout(h http.Handler) http.Handler {
	return http.TimeoutHandler(h, time.Second*5, "Application has timed out.")
}
