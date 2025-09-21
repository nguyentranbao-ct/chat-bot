package tmplx

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/goccy/go-json"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

var (
	ErrRenderTemplate = errors.New("tmplx: render error")
	ErrParseTemplate  = errors.New("tmplx: parse error")
)

type Template struct {
	tmpl *template.Template
}

type Options struct {
	validate ValidateFunc
	testData any
	funcs    template.FuncMap
}

type Option func(*Options) error

type ValidateFunc func(*bytes.Buffer) error

// defaultFuncs returns the default template functions
func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"quote":          quoteFunc,
		"default":        defaultFunc,
		"json":           jsonFunc,
		"hasSuffix":      hasSuffix,
		"hasPrefix":      hasPrefix,
		"regexMatch":     regexMatch,
		"jsonGet":        jsonGet,
		"encodeUrlQuery": encodeUrlQuery,
	}
}

// WithTemplateFunc adds a single custom template function
func WithTemplateFunc(name string, fn any) Option {
	return func(t *Options) error {
		t.funcs[name] = fn
		return nil
	}
}

// WithValidate adds validation using test data
func WithValidate(testData any, validateFn ValidateFunc) Option {
	return func(t *Options) error {
		t.validate = validateFn
		t.testData = testData
		return nil
	}
}

func MustParse(name string, text string, opts ...Option) *Template {
	t, err := Parse(name, text, opts...)
	if err != nil {
		panic(err)
	}
	return t
}

// Parse creates a new Template with the given name and text, applying any options
func Parse(name string, text string, args ...Option) (*Template, error) {
	opts := &Options{
		funcs: defaultFuncs(),
	}
	for _, arg := range args {
		if err := arg(opts); err != nil {
			return nil, err
		}
	}

	tmpl, err := template.New(name).
		Option("missingkey=zero").
		Funcs(opts.funcs).
		Parse(text)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseTemplate, err)
	}

	t := &Template{
		tmpl: tmpl,
	}
	if opts.validate != nil {
		if err := t.validate(opts.testData, opts.validate); err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (t *Template) validate(data any, validate ValidateFunc) error {
	buf := new(bytes.Buffer)
	if err := t.tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := validate(buf); err != nil {
		return fmt.Errorf("validate template: %w", err)
	}
	return nil
}

func (t *Template) Render(data any) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := t.tmpl.Execute(buf, data); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRenderTemplate, err)
	}
	return buf, nil
}

func hasSuffix(a, b any) bool {
	s1 := cast.ToString(a)
	s2 := cast.ToString(b)
	return strings.HasSuffix(s1, s2)
}

func hasPrefix(a, b any) bool {
	s1 := cast.ToString(a)
	s2 := cast.ToString(b)
	return strings.HasPrefix(s1, s2)
}

func quoteFunc(s string) (string, error) {
	return jsonFunc(s)
}

func defaultFunc(def any, value any) any {
	if value != nil && value != "" {
		return value
	}
	return def
}

func jsonFunc(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func regexMatch(in string, expr string) (bool, error) {
	r, err := regexp.Compile(expr)
	if err != nil {
		return false, err
	}
	return r.MatchString(in), nil
}

func jsonGet(path string, raw string) string {
	return gjson.Get(raw, path).String()
}

func encodeUrlQuery(queries ...any) string {
	query := url.Values{}
	for i := 0; i < len(queries); i += 2 {
		value := ""
		if i+1 < len(queries) {
			value = cast.ToString(queries[i+1])
		}
		query.Add(cast.ToString(queries[i]), value)
	}
	return query.Encode()
}

var fieldsRegexp = regexp.MustCompile(`{{[^{}]*\.(\w+)[^{}]*}}`)

func ExtractFields(content string) []string {
	matches := fieldsRegexp.FindAllStringSubmatch(content, -1)
	fields := make([]string, 0)
	dict := make(map[string]struct{})
	for _, match := range matches {
		if len(match) == 2 && match[1] != "" {
			if _, ok := dict[match[1]]; !ok {
				fields = append(fields, match[1])
				dict[match[1]] = struct{}{}
			}
		}
	}
	return fields
}
