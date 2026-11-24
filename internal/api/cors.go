package api

import "net/http"

// corsMaxAge lets a browser cache a preflight answer for ten minutes, in seconds.
const corsMaxAge = "600"

// allowCORS lets listed browser origins call the API. Other origins get no CORS
// header, so the browser refuses to hand them the response. A preflight from a listed
// origin is answered here with 204, before authentication: browsers send no
// Authorization header on OPTIONS. Every real request still needs the bearer token.
func allowCORS(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")
		if origin == "" || !allowed[origin] {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			w.Header().Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
