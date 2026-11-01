package cors

import (
	"net/http"
	"strconv"
	"strings"
)

type Config struct {
	Origins       []string
	Methods       string
	Headers       string
	ExposeHeaders string
	Credentials   bool
	MaxAge        string
}

func Middleware(cfg Config) func(http.Handler) http.Handler {
	originSet := map[string]bool{}
	for _, o := range cfg.Origins {
		originSet[strings.TrimSpace(o)] = true
	}
	allowAll := originSet["*"]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					if cfg.Credentials {
						w.Header().Set("Access-Control-Allow-Origin", origin)
					} else {
						w.Header().Set("Access-Control-Allow-Origin", "*")
					}
				} else if originSet[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				if cfg.Credentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				if cfg.ExposeHeaders != "" {
					w.Header().Set("Access-Control-Expose-Headers", cfg.ExposeHeaders)
				}
				if cfg.MaxAge != "" {
					w.Header().Set("Access-Control-Max-Age", cfg.MaxAge)
				}
			}

			if r.Method == http.MethodOptions {
				if origin != "" {
					if cfg.Methods != "" {
						w.Header().Set("Access-Control-Allow-Methods", cfg.Methods)
					}
					if cfg.Headers != "" {
						w.Header().Set("Access-Control-Allow-Headers", cfg.Headers)
					} else {
						if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
							w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
						}
					}
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (c Config) MaxAgeSeconds() int {
	if c.MaxAge == "" {
		return 3600
	}
	if n, err := strconv.Atoi(c.MaxAge); err == nil {
		return n
	}
	return 3600
}
