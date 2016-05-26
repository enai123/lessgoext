package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/lessgo/lessgo"
)

type (
	// JWTConfig defines the config for JWT auth middleware.
	JWTConfig struct {
		// SigningKey is the key to validate token.
		// Required.
		SigningKey string `json:"signing_key"`

		// SigningMethod is used to check token signing method.
		// Optional, with default value as `HS256`.
		SigningMethod string `json:"signing_method"`

		// ContextKey is the key to be used for storing user information from the
		// token into context.
		// Optional, with default value as `user`.
		ContextKey string `json:"context_key"`

		// Extractor is a function that extracts token from the request.
		// Optional, with default values as `JWTFromHeader`.
		Extractor JWTExtractor
	}

	// JWTExtractor defines a function that takes `lessgo.Context` and returns either
	// a token or an error.
	JWTExtractor func(*lessgo.Context) (string, error)
)

const (
	bearer = "Bearer"
)

// Algorithims
const (
	AlgorithmHS256 = "HS256"
)

var (
	// DefaultJWTConfig is the default JWT auth middleware config.
	DefaultJWTConfig = JWTConfig{
		SigningMethod: AlgorithmHS256,
		ContextKey:    "user",
		Extractor:     JWTFromHeader,
	}
)

// For valid token, it sets the user in context and calls next handler.
// For invalid token, it sends "401 - Unauthorized" response.
// For empty or invalid `Authorization` header, it sends "400 - Bad Request".
//
// See https://jwt.io/introduction
// JWTWithConfig returns a JWT auth middleware from config.
var JWTWithConfig = lessgo.ApiMiddleware{
	Name:   "JWTWithConfig",
	Desc:   `JWT基本的第三方授权中间件，使用前请先在源码配置处理函数。`,
	Config: DefaultJWTConfig,
	Middleware: func(confObject interface{}) lessgo.MiddlewareFunc {
		config := confObject.(JWTConfig)
		// Defaults
		if len(config.SigningKey) == 0 {
			panic("jwt middleware requires signing key")
		}
		if config.SigningMethod == "" {
			config.SigningMethod = DefaultJWTConfig.SigningMethod
		}
		if config.ContextKey == "" {
			config.ContextKey = DefaultJWTConfig.ContextKey
		}
		if config.Extractor == nil {
			config.Extractor = DefaultJWTConfig.Extractor
		}

		return func(next lessgo.HandlerFunc) lessgo.HandlerFunc {
			return func(c *lessgo.Context) error {
				auth, err := config.Extractor(c)
				if err != nil {
					return lessgo.NewHTTPError(http.StatusBadRequest, err.Error())
				}
				token, err := jwt.Parse(auth, func(t *jwt.Token) (interface{}, error) {
					// Check the signing method
					if t.Method.Alg() != config.SigningMethod {
						return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
					}
					return []byte(config.SigningKey), nil

				})
				if err == nil && token.Valid {
					// Store user information from token into context.
					c.Set(config.ContextKey, token)
					return next(c)
				}
				return lessgo.ErrUnauthorized
			}
		}
	},
}

// JWTFromHeader is a `JWTExtractor` that extracts token from the `Authorization` request
// header.
func JWTFromHeader(c *lessgo.Context) (string, error) {
	auth := c.Request().Header.Get(lessgo.HeaderAuthorization)
	l := len(bearer)
	if len(auth) > l+1 && auth[:l] == bearer {
		return auth[l+1:], nil
	}
	return "", errors.New("empty or invalid jwt in authorization header")
}

// JWTFromQuery returns a `JWTExtractor` that extracts token from the provided query
// parameter.
func JWTFromQuery(param string) JWTExtractor {
	return func(c *lessgo.Context) (string, error) {
		token := c.QueryParam(param)
		if token == "" {
			return "", errors.New("empty jwt in query param")
		}
		return token, nil
	}
}
