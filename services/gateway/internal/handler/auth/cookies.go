package auth

import "net/http"

const (
	cookieAccessToken  = "access_token"
	cookieRefreshToken = "refresh_token"
	accessTokenMaxAge  = 900               // 15 minutes
	refreshTokenMaxAge = 60 * 60 * 24 * 30 // 30 days
)

func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieAccessToken,
		Value:    accessToken,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   accessTokenMaxAge,
		Path:     "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cookieRefreshToken,
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   refreshTokenMaxAge,
		Path:     "/auth/session",
	})
}

func clearAuthCookies(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieAccessToken,
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   -1,
		Path:     "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cookieRefreshToken,
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   -1,
		Path:     "/auth/session",
	})
}
