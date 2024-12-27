package utils

import (
	"fmt"
	"net/http"

	"github.com/frhan23/chat-go/middleware"

	"github.com/golang-jwt/jwt/v5"
)

func ExtractToken(r *http.Request) (jwt.MapClaims, error) {
	claims, ok := r.Context().Value(middleware.TokenKey).(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("token not found in context")
	}
	return claims, nil
}
