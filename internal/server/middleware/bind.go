package middleware

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/cstockton/go-conv"
	"github.com/golang-jwt/jwt/v4"

	"github.com/labstack/echo/v4"
)

// BindAndValidate bind request context and validate request struct.
// Bind includes request body, params, query, headers and standard jwt claims.
// Validate request struct, response bad request with error message if the request is invalid.
func BindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return err
	}

	if err := bindHeader(c.Request().Header, req); err != nil {
		return err
	}

	if err := bindStandardJwt(c, req); err != nil {
		return err
	}

	if err := bindRegisteredJwt(c, req); err != nil {
		return err
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return nil
}

func extractJwtToken(c echo.Context) (*jwt.Token, error) {
	data := c.Get("user")
	if data == nil {
		return nil, nil
	}

	token, ok := data.(*jwt.Token)
	if !ok {
		return nil, fmt.Errorf("cannot cast jwt token: %#v", data)
	}

	return token, nil
}

func extractJwtStandardClaims(token *jwt.Token) (*jwt.StandardClaims, error) {
	claims, ok := token.Claims.(*jwt.StandardClaims)
	if !ok {
		return nil, fmt.Errorf("cannot cast jwt standard claims: %+v", token.Claims)
	}

	return claims, nil
}

func extractJwtRegisteredClaims(token *jwt.Token) (*jwt.RegisteredClaims, error) {
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("cannot cast jwt registered claims: %+v", token.Claims)
	}

	return claims, nil
}

func GetUserID(c echo.Context) string {
	token, _ := extractJwtToken(c)
	if token == nil {
		return ""
	}

	standard, _ := extractJwtStandardClaims(token)
	if standard != nil {
		return standard.Subject
	}

	registered, _ := extractJwtRegisteredClaims(token)
	if registered != nil {
		return registered.Subject
	}

	return ""
}

// bindStandardJwt use standard jwt claims to decode jwt to struct by tag `jwt:"payloadField"`
func bindStandardJwt(c echo.Context, dst interface{}) error {
	token, _ := extractJwtToken(c)
	if token == nil {
		return nil
	}

	claims, _ := extractJwtStandardClaims(token)
	if claims == nil {
		return nil
	}

	getValueFn := func(tagValue string) (interface{}, error) {
		var value interface{}
		switch tagValue {
		case "sub":
			value = claims.Subject
		case "iss":
			value = claims.Issuer
		case "aud":
			value = claims.Audience
		case "jti":
			value = claims.Id
		case "exp":
			value = claims.ExpiresAt
		case "iat":
			value = claims.IssuedAt
		case "nbf":
			value = claims.NotBefore
		default:
			return nil, fmt.Errorf("binding jwt field %s is not supported", tagValue)
		}
		return value, nil
	}

	return bindStruct(dst, "jwt", getValueFn)
}

// bindRegisteredJwt use registered jwt claims to decode jwt to struct by tag `jwt:"payloadField"`
func bindRegisteredJwt(c echo.Context, dst interface{}) error {
	token, _ := extractJwtToken(c)
	if token == nil {
		return nil
	}

	claims, _ := extractJwtRegisteredClaims(token)
	if claims == nil {
		return nil
	}

	getValueFn := func(tagValue string) (interface{}, error) {
		var value interface{}
		switch tagValue {
		case "sub":
			value = claims.Subject
		case "iss":
			value = claims.Issuer
		case "aud":
			value = strings.Join(claims.Audience, ";")
		case "jti":
			value = claims.ID
		case "exp":
			value = claims.ExpiresAt.Unix()
		case "iat":
			value = claims.IssuedAt.Unix()
		case "nbf":
			value = claims.NotBefore.Unix()
		default:
			return nil, fmt.Errorf("binding jwt field %s is not supported", tagValue)
		}
		return value, nil
	}

	return bindStruct(dst, "jwt", getValueFn)
}

// bindHeader decode http header to struct by tag `header:"<header_name>"`
// out must be a pointer to a struct
func bindHeader(header http.Header, dst interface{}) error {
	getValueFn := func(tagValue string) (interface{}, error) {
		return header.Get(tagValue), nil
	}

	return bindStruct(dst, "header", getValueFn)
}

// bindStruct decode to struct by custom tag `tagName:"tagValue"`
// dst must be a pointer to a struct
func bindStruct(dst interface{}, tagName string, getValueFn func(tagValue string) (interface{}, error)) error {
	ptr := reflect.ValueOf(dst)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("non-pointer passed to Unmarshal")
	}

	indirect := reflect.Indirect(ptr)
	structType := indirect.Type()
	elemZero := reflect.Zero(structType)

	numField := elemZero.NumField()
	for i := 0; i < numField; i++ {
		structField := structType.Field(i)
		tagValue := structField.Tag.Get(tagName)
		if tagValue == "-" || tagValue == "" {
			continue
		}

		field := indirect.Field(i)
		value, err := getValueFn(tagValue)
		if err != nil {
			return err
		}
		if err := conv.Infer(field, value); err != nil {
			return fmt.Errorf("cannot parse %s.%s as %s from: %#v / %s",
				structType.Name(), structField.Name, field.Type(), value, err)
		}
	}

	return nil
}
