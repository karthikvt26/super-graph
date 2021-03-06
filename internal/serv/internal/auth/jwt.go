package auth

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dosco/super-graph/core"
)

const (
	authHeader     = "Authorization"
	jwtAuth0   int = iota + 1
)

func JwtHandler(ac *Auth, next http.Handler) (http.HandlerFunc, error) {
	var key interface{}
	var jwtProvider int

	cookie := ac.Cookie

	if ac.JWT.Provider == "auth0" {
		jwtProvider = jwtAuth0
	}

	secret := ac.JWT.Secret
	publicKeyFile := ac.JWT.PubKeyFile

	switch {
	case len(secret) != 0:
		key = []byte(secret)

	case len(publicKeyFile) != 0:
		kd, err := ioutil.ReadFile(publicKeyFile)
		if err != nil {
			return nil, err
		}

		switch ac.JWT.PubKeyType {
		case "ecdsa":
			key, err = jwt.ParseECPublicKeyFromPEM(kd)

		case "rsa":
			key, err = jwt.ParseRSAPublicKeyFromPEM(kd)

		default:
			key, err = jwt.ParseECPublicKeyFromPEM(kd)

		}

		if err != nil {
			return nil, err
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var tok string

		if len(cookie) != 0 {
			ck, err := r.Cookie(cookie)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			tok = ck.Value
		} else {
			ah := r.Header.Get(authHeader)
			if len(ah) < 10 {
				next.ServeHTTP(w, r)
				return
			}
			tok = ah[7:]
		}

		token, err := jwt.ParseWithClaims(tok, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
			return key, nil
		})

		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if claims, ok := token.Claims.(*jwt.StandardClaims); ok {
			ctx := r.Context()

			if jwtProvider == jwtAuth0 {
				sub := strings.Split(claims.Subject, "|")
				if len(sub) != 2 {
					ctx = context.WithValue(ctx, core.UserIDProviderKey, sub[0])
					ctx = context.WithValue(ctx, core.UserIDKey, sub[1])
				}
			} else {
				ctx = context.WithValue(ctx, core.UserIDKey, claims.Subject)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		next.ServeHTTP(w, r)
	}, nil
}
