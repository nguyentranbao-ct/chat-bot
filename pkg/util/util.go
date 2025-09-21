package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
)

func PanicOnError(msg string, err ...error) error {
	for _, e := range err {
		if e != nil {
			return fmt.Errorf("%s: %v", msg, e)
		}
	}
	return nil
}

func ConvertListE[A any, B any](listA []A, convert func(A) (B, error)) ([]B, error) {
	listB := make([]B, len(listA))
	for i, a := range listA {
		b, err := convert(a)
		if err != nil {
			return nil, err
		}
		listB[i] = b
	}

	return listB, nil
}

func ConvertList[A any, B any](listA []A, convert func(A) B) []B {
	listB := make([]B, len(listA))
	for i, a := range listA {
		listB[i] = convert(a)
	}

	return listB
}

func SliceIncludes[T comparable](values []T, value T) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func TranscodeJSON(in, out interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(in); err != nil {
		return err
	}
	if err := json.NewDecoder(buf).Decode(out); err != nil {
		return err
	}
	return nil
}

type nopLogger struct{}

func (nopLogger) Errorf(string, ...interface{}) {}
func (nopLogger) Warnf(string, ...interface{})  {}
func (nopLogger) Debugf(string, ...interface{}) {}

func NewRestyClient() *resty.Client {
	c := resty.
		New().
		SetRetryCount(3).
		SetLogger(nopLogger{}).
		SetTimeout(10 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			retry, _ := retryablehttp.DefaultRetryPolicy(r.Request.Context(), r.RawResponse, err)
			return retry
		})
	c.JSONMarshal = json.Marshal
	c.JSONUnmarshal = json.Unmarshal
	return c
}

// Ptr returns pointer of any value.
func Ptr[T any](t T) *T {
	return &t
}

// Val returns value if pointer is not null, otherwise it returns zero.
func Val[T any](t *T) T {
	if t != nil {
		return *t
	}

	var def T
	return def
}

// deep clone struct
func Clone[T any](src T) (T, error) {
	var out T
	data, err := json.Marshal(src)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(data, &out)
	return out, err
}

func GetHistogramVec(name string, labels ...string) (*prometheus.HistogramVec, error) {
	metrics := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: name,
		Buckets: []float64{
			0.0005,
			0.001, // 1ms
			0.002,
			0.005,
			0.01, // 10ms
			0.02,
			0.05,
			0.1, // 100 ms
			0.2,
			0.5,
			1.0, // 1s
			2.0,
			5.0,
			10.0, // 10s
		},
	}, labels)
	if err := prometheus.Register(metrics); err != nil {
		var registeredErr prometheus.AlreadyRegisteredError
		if ok := errors.As(err, &registeredErr); ok {
			metrics, ok := registeredErr.ExistingCollector.(*prometheus.HistogramVec)
			if ok {
				return metrics, nil
			}
		}
		return nil, fmt.Errorf("register: %w %T", err, err)
	}

	return metrics, nil
}
