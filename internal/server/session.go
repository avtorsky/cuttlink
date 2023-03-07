package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	sessionCookieName   = "cluid"
	sessionCookieMaxAge = 86400
	uuidHashKey         = "secret42"
)

func generateCookieToken() string {
	id, err := uuid.New().MarshalBinary()
	if err != nil {
		return ""
	}
	h := hmac.New(sha256.New, []byte(uuidHashKey))
	h.Write(id)
	sign := h.Sum(nil)
	token := append(id, sign...)

	return hex.EncodeToString(token)
}

func validateCookieToken(token string) bool {
	data, err := hex.DecodeString(token)
	if err != nil || len(data) < 32 {
		return false
	}
	id := data[:16]
	h := hmac.New(sha256.New, []byte(uuidHashKey))
	h.Write(id)
	sign := h.Sum(nil)

	return hmac.Equal(sign, data[:16])
}

func cookieAuthentication() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cookie, err := ctx.Cookie(sessionCookieName)
		if validateCookieToken(cookie) {
			http.Error(ctx.Writer, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			token := generateCookieToken()
			ctx.SetCookie(sessionCookieName, token, sessionCookieMaxAge, "/", "localhost", false, true)
			ctx.Next()
		}
		ctx.Next()
	}
}

func getUUID(ctx *gin.Context) (string, error) {
	token, err := ctx.Cookie(sessionCookieName)
	if err != nil {
		h := ctx.Writer.Header().Values("Set-Cookie")[0]
		c := strings.Split(h, ";")[0]
		t := strings.Split(c, "=")[1]
		d, _ := hex.DecodeString(t)
		i, _ := uuid.FromBytes(d[:16])
		return i.String(), nil
	}
	data, _ := hex.DecodeString(token)
	id, _ := uuid.FromBytes(data[:16])
	return id.String(), nil
}
