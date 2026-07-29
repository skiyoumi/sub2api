package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const registrationDeviceCookieName = "sub2api_registration_device"
const registrationDeviceCookieMaxAge = 365 * 24 * 60 * 60

func (h *AuthHandler) ensureRegistrationDevice(c *gin.Context) string {
	if h == nil || h.cfg == nil || strings.TrimSpace(h.cfg.JWT.Secret) == "" {
		return ""
	}
	if cookie, err := c.Request.Cookie(registrationDeviceCookieName); err == nil {
		if id, ok := verifyRegistrationDeviceCookie(cookie.Value, h.cfg.JWT.Secret); ok {
			return id
		}
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	id := base64.RawURLEncoding.EncodeToString(buf)
	value := id + "." + registrationDeviceSignature(id, h.cfg.JWT.Secret)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     registrationDeviceCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   registrationDeviceCookieMaxAge,
		HttpOnly: true,
		Secure:   isRequestHTTPS(c),
		SameSite: http.SameSiteLaxMode,
	})
	return id
}

func verifyRegistrationDeviceCookie(value, secret string) (string, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	expected := registrationDeviceSignature(parts[0], secret)
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return "", false
	}
	if _, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		return "", false
	}
	return parts[0], true
}

func registrationDeviceSignature(id, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
