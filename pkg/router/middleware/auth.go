package middleware

// func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.Header.Get("Authorization") == "" {
// 			w.Header().Set("Content-Type", "application/json")
// 			w.WriteHeader(http.StatusUnauthorized)
// 			json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
// 			return
// 		}
// 		next(w, r)
// 	}
// }
