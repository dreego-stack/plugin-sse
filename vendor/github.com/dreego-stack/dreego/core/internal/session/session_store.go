package session

import "net/http"

type Store interface {
	Get(r *http.Request, key string) (string, error)
	Set(w http.ResponseWriter, r *http.Request, key, value string, opts *Options) error
	Delete(w http.ResponseWriter, r *http.Request, key string) error
	Destroy(w http.ResponseWriter, r *http.Request) error
}

type Options struct {
	MaxAge   int
	Secure   bool
	HttpOnly bool
	Path     string
	Encrypt  bool
}

type storeValidator interface {
	Validate() error
}
