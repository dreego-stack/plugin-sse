package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

const minSecretLen = 32
const MaxCookieSize = 4096

var ErrSessionTooLarge = errors.New("dreego: encoded session state exceeds cookie size limit")
var ErrCookiePathOverride = errors.New("dreego: per-call cookie path overrides are not allowed; configure CookiePolicy.Path")

type randReader interface {
	Read(p []byte) (n int, err error)
}

type CookieStore struct {
	mu             sync.RWMutex
	secret         []byte
	name           string
	rand           randReader
	policy         CookiePolicy
	trustedProxies map[string]bool
}

type CookiePolicy struct {
	SameSite http.SameSite
	Secure   bool
	HttpOnly bool
	Path     string
	Encrypt  bool
}

func NewCookieStore(secret []byte) *CookieStore {
	if len(secret) < minSecretLen {
		panic("dreego: session secret must be at least 32 bytes")
	}
	return &CookieStore{
		secret: secret,
		name:   "dreego_session",
		rand:   randReader(rand.Reader),
		policy: CookiePolicy{
			SameSite: http.SameSiteLaxMode,
			HttpOnly: true,
			Path:     "/",
		},
		trustedProxies: map[string]bool{},
	}
}

func (s *CookieStore) SetCookiePolicy(p CookiePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.SameSite != 0 {
		s.policy.SameSite = p.SameSite
	}
	if p.Secure {
		s.policy.Secure = true
	}
	if p.HttpOnly {
		s.policy.HttpOnly = true
	}
	if p.Path != "" {
		s.policy.Path = p.Path
	}
	if p.Encrypt {
		s.policy.Encrypt = true
	}
}

func (s *CookieStore) SetTrustedProxies(addrs []string) {
	m := map[string]bool{}
	for _, a := range addrs {
		m[a] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trustedProxies = m
}

func (s *CookieStore) Name() string { return s.name }

func (s *CookieStore) TrustedProxies() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proxies := make(map[string]bool, len(s.trustedProxies))
	for k, v := range s.trustedProxies {
		proxies[k] = v
	}
	return proxies
}

func (s *CookieStore) Validate() error {
	if len(s.secret) < minSecretLen {
		return errors.New("dreego: session secret must be at least 32 bytes")
	}
	return nil
}

func (s *CookieStore) Get(r *http.Request, key string) (string, error) {
	m, err := s.load(r)
	if err != nil {
		return "", err
	}
	return m[key], nil
}

func (s *CookieStore) Set(w http.ResponseWriter, r *http.Request, key, value string, opts *Options) error {
	if err := s.validatePathOverride(opts); err != nil {
		return err
	}
	m, err := s.load(r)
	if err != nil {
		m = map[string]string{}
	}
	next := make(map[string]string, len(m)+1)
	for k, v := range m {
		next[k] = v
	}
	if value == "" {
		delete(next, key)
	} else {
		next[key] = value
	}
	encoded, err := s.sign(next, s.resolveEncrypt(opts))
	if err != nil {
		return err
	}
	if len(encoded) > MaxCookieSize {
		return ErrSessionTooLarge
	}
	*r = *r.WithContext(context.WithValue(r.Context(), ctxKey{}, next))
	http.SetCookie(w, &http.Cookie{
		Name:     s.name,
		Value:    encoded,
		MaxAge:   opt(opts, func(o *Options) int { return o.MaxAge }),
		Secure:   s.resolveSecure(r, opts),
		HttpOnly: s.resolveHttpOnly(opts),
		SameSite: s.resolveSameSite(),
		Path:     s.resolvePath(opts),
	})
	return nil
}

func (s *CookieStore) load(r *http.Request) (map[string]string, error) {
	if m, ok := r.Context().Value(ctxKey{}).(map[string]string); ok {
		return m, nil
	}
	ck, err := r.Cookie(s.name)
	if err != nil {
		return map[string]string{}, nil
	}
	data, ok := s.verify(ck.Value)
	if !ok {
		return nil, errors.New("dreego: session cookie failed integrity verification")
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, errors.New("dreego: session cookie payload is not valid JSON")
	}
	return m, nil
}

func (s *CookieStore) Delete(w http.ResponseWriter, r *http.Request, key string) error {
	return s.Set(w, r, key, "", s.policyToOptions(r))
}

func (s *CookieStore) Destroy(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     s.name,
		Value:    "",
		MaxAge:   -1,
		Secure:   s.resolveSecure(r, nil),
		HttpOnly: s.resolveHttpOnly(nil),
		SameSite: s.resolveSameSite(),
		Path:     s.resolvePath(nil),
	})
	*r = *r.WithContext(context.WithValue(r.Context(), ctxKey{}, map[string]string{}))
	return nil
}

func (s *CookieStore) sign(m map[string]string, encrypt bool) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	key := deriveKeys(s.secret)
	if encrypt {
		enc, err := encryptPayload(key.enc, data, s.rand)
		if err != nil {
			return "", err
		}
		data = append([]byte{encMarker}, enc...)
	}
	mac := hmac.New(sha256.New, key.sig)
	mac.Write(data)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(append(sig, data...)), nil
}

func (s *CookieStore) verify(value string) ([]byte, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, false
	}
	if len(decoded) < sha256.Size {
		return nil, false
	}
	sig := decoded[:sha256.Size]
	data := decoded[sha256.Size:]
	key := deriveKeys(s.secret)
	mac := hmac.New(sha256.New, key.sig)
	mac.Write(data)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return nil, false
	}
	if len(data) > 0 && data[0] == encMarker {
		return decryptPayload(key.enc, data[1:])
	}
	return data, true
}
