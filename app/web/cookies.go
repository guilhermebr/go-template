package web

import (
	"net/http"
	"time"

	gweb "go-template/gateways/web"
)

const (
	CookieToken       = "token"
	CookieUserID      = "user_id"
	CookieUserEmail   = "user_email"
	CookieAccountType = "account_type"
)

// Cookie management methods

func (m *AuthMiddleware) setAuthCookies(w http.ResponseWriter, resp *gweb.AuthResponse) {
	maxAge := m.cookieMaxAge

	// Don't set domain for localhost in development
	var domain string
	if m.cookieDomain != "localhost" && m.cookieDomain != "" {
		domain = m.cookieDomain
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieToken,
		Value:    resp.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Domain:   domain,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     CookieUserID,
		Value:    resp.User.ID.String(),
		Path:     "/",
		HttpOnly: false,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Domain:   domain,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     CookieUserEmail,
		Value:    resp.User.Email,
		Path:     "/",
		HttpOnly: false,
		Secure:   m.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Domain:   domain,
	})
}

func (m *AuthMiddleware) clearAuthCookies(w http.ResponseWriter) {
	cookieNames := []string{CookieToken, CookieUserID, CookieUserEmail}

	// Don't set domain for localhost in development
	var domain string
	if m.cookieDomain != "localhost" && m.cookieDomain != "" {
		domain = m.cookieDomain
	}

	for _, name := range cookieNames {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == CookieToken,
			Secure:   m.cookieSecure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			Domain:   domain,
		})
	}
}

func getCookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
