package middleware

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	validate := validator.New()

	commonTags := []string{
		"json",
		"param",
		"query",
		"header",
	}

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		for _, tag := range commonTags {
			name := strings.SplitN(fld.Tag.Get(tag), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return ""
	})

	validate.RegisterValidation("urls", func(fl validator.FieldLevel) bool {
		slice, ok := fl.Field().Interface().([]string)
		if !ok {
			return false
		}
		for _, s := range slice {
			err := validate.Var(s, "url")
			if err != nil {
				return false
			}
		}
		return true
	})

	v := &Validator{
		validate: validate,
	}

	return v
}

func (v *Validator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}
