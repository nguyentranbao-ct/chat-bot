package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestBindHeader(t *testing.T) {
	type args struct {
		header map[string]string
		out    interface{}
	}

	type normalCase struct {
		App     string `header:"app"`
		Service string `header:"service"`

		Non   string `header:"-"`
		Empty bool
	}

	type complexCase struct {
		Nine              int64   `header:"nine"`
		ThousandAndSeven  uint64  `header:"thousand-and-seven"`
		NegativeThirtyTwo int64   `header:"negative-thirty-two"`
		HundredPointSix   float32 `header:"hundred-point-six"`
		Rose              string  `header:"rose"`
	}

	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr error
	}{
		{
			name: "normal bind header",
			args: args{
				header: map[string]string{
					"app":     "chotot",
					"service": "uac-consumer",
					"non":     "non",
					"empty":   "empty",
				},
				out: new(normalCase),
			},
			want: &normalCase{
				App:     "chotot",
				Service: "uac-consumer",
				Non:     "",
				Empty:   false,
			},
			wantErr: nil,
		},
		{
			name: "complex bind header",
			args: args{
				header: map[string]string{
					"nine":                "9",
					"thousand-and-seven":  "1007",
					"negative-thirty-two": "-32",
					"hundred-point-six":   "100.6",
					"rose":                "rose",
				},
				out: new(complexCase),
			},
			want: &complexCase{
				Nine:              9,
				ThousandAndSeven:  1007,
				NegativeThirtyTwo: -32,
				HundredPointSix:   100.6,
				Rose:              "rose",
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			for k, v := range tt.args.header {
				header.Set(k, v)
			}
			err := bindHeader(header, tt.args.out)
			assert.EqualValues(t, err, tt.wantErr)
			assert.EqualValues(t, tt.want, tt.args.out)
		})
	}
}

func TestBindJwt(t *testing.T) {
	type args struct {
		claims *jwt.StandardClaims
		out    interface{}
	}

	type normalClaims struct {
		UserID    string  `jwt:"sub"`
		Issuer    string  `jwt:"iss"`
		ID        string  `jwt:"jti"`
		Audience  string  `jwt:"aud"`
		IssuedAt  uint    `jwt:"iat"`
		NotBefore string  `jwt:"nbf"`
		Exp       float64 `jwt:"exp"`
	}

	type invalidClaims struct {
		UserID    int  `jwt:"sub"`
		ExpiresAt int  `jwt:"exp"`
		Audience  bool `jwt:"aud"`
	}

	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr error
	}{
		{
			name: "fully jwt binding",
			args: args{
				claims: &jwt.StandardClaims{
					Subject:   "123",
					Issuer:    "carousell",
					Id:        "idc",
					Audience:  "audienc3",
					IssuedAt:  108924701923,
					NotBefore: 1290465712,
					ExpiresAt: 9042,
				},
				out: new(normalClaims),
			},
			want: &normalClaims{
				UserID:    "123",
				Issuer:    "carousell",
				ID:        "idc",
				Audience:  "audienc3",
				IssuedAt:  108924701923,
				NotBefore: "1290465712",
				Exp:       9042,
			},
			wantErr: nil,
		},
		{
			name: "binding with wrong type",
			args: args{
				claims: &jwt.StandardClaims{
					Subject:   "90174",
					Issuer:    "carousell",
					Audience:  "not boolean",
					ExpiresAt: 9042,
				},
				out: new(invalidClaims),
			},
			want: &invalidClaims{
				UserID:    90174,
				ExpiresAt: 9042,
				Audience:  false,
			},
			wantErr: fmt.Errorf(`cannot parse invalidClaims.Audience as bool from: "not boolean" / cannot parse string with len 11 as bool`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user", &jwt.Token{
				Claims: tt.args.claims,
			})
			err := bindStandardJwt(c, tt.args.out)
			assert.EqualValues(t, tt.wantErr, err)
			assert.EqualValues(t, tt.want, tt.args.out)
		})
	}
}
