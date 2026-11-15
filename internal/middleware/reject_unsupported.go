package middleware

import "net/http"

func RejectUnsupported(header, errMsg string) func(http.Handler) http.Handler {

	canonicalHeader := http.CanonicalHeaderKey(header)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := r.Header[canonicalHeader]; ok {
				http.Error(w, errMsg, http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
