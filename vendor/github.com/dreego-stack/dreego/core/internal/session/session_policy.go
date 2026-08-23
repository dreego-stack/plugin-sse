package session

import (
	"net"
	"net/http"
)

func (s *CookieStore) resolveSecure(r *http.Request, opts *Options) bool {
	if opts != nil && opts.Secure {
		return true
	}
	s.mu.RLock()
	secure := s.policy.Secure
	proxies := s.trustedProxies
	s.mu.RUnlock()
	if secure {
		return true
	}
	return IsTLS(r, proxies)
}

func (s *CookieStore) resolveHttpOnly(opts *Options) bool {
	if opts != nil && opts.HttpOnly {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy.HttpOnly
}

func (s *CookieStore) resolveSameSite() http.SameSite {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.policy.SameSite != 0 {
		return s.policy.SameSite
	}
	return http.SameSiteLaxMode
}

func (s *CookieStore) resolvePath(opts *Options) string {
	if opts != nil && opts.Path != "" {
		return opts.Path
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.policy.Path != "" {
		return s.policy.Path
	}
	return "/"
}

func (s *CookieStore) validatePathOverride(opts *Options) error {
	if opts == nil || opts.Path == "" {
		return nil
	}
	if opts.Path != s.resolvePath(nil) {
		return ErrCookiePathOverride
	}
	return nil
}

func (s *CookieStore) resolveEncrypt(opts *Options) bool {
	if opts != nil && opts.Encrypt {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy.Encrypt
}

func (s *CookieStore) policyToOptions(r *http.Request) *Options {
	s.mu.RLock()
	secure := s.policy.Secure
	httpOnly := s.policy.HttpOnly
	path := s.policy.Path
	encrypt := s.policy.Encrypt
	proxies := s.trustedProxies
	s.mu.RUnlock()
	return &Options{
		Secure:   secure || IsTLS(r, proxies),
		HttpOnly: httpOnly,
		Path:     path,
		Encrypt:  encrypt,
	}
}

func IsTLS(r *http.Request, trustedProxies map[string]bool) bool {
	if r.TLS != nil {
		return true
	}
	if len(trustedProxies) == 0 {
		return false
	}
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if !trustedProxies[host] {
		return false
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
