package middleware

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"

	"github.com/labstack/echo/v4"
)

// WrapHandler wraps echo handler with extra binding and validation
func WrapHandler(f interface{}) echo.HandlerFunc {
	handler, err := wrapHandler(f)
	if err != nil {
		panic(err)
	}

	return handler
}

func wrapHandler(f interface{}) (echo.HandlerFunc, error) {
	fTyp := reflect.TypeOf(f)
	fVal := reflect.ValueOf(f)

	if fVal.Kind() != reflect.Func {
		return nil, fmt.Errorf("invalid function passed to wrap handler: %v", fVal)
	}
	fName := runtime.FuncForPC(fVal.Pointer()).Name()

	numIn := fTyp.NumIn()
	if fTyp.NumIn() != 2 {
		return nil, fmt.Errorf("[%s] invalid function arguments length: %d", fName, numIn)
	}

	ctxInterface := reflect.TypeOf((*echo.Context)(nil)).Elem()
	if !fTyp.In(0).Implements(ctxInterface) {
		return nil, fmt.Errorf("[%s] first argument must has type echo.Context", fName)
	}

	secKind := fTyp.In(1).Kind()
	if secKind != reflect.Struct {
		return nil, fmt.Errorf("[%s] second argument must has type struct: %v", fName, secKind)
	}

	numOut := fTyp.NumOut()
	if numOut < 1 || numOut > 2 {
		return nil, fmt.Errorf("[%s] invalid function returns length: %d", fName, numOut)
	}

	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	errorIndex := numOut - 1
	lastReturnType := fTyp.Out(errorIndex)
	if !lastReturnType.Implements(errorInterface) {
		return nil, fmt.Errorf("[%s] last return argument must has type error: %v", fName, lastReturnType)
	}

	reqType := fTyp.In(1)

	handler := func(c echo.Context) error {
		req := reflect.New(reqType)
		if err := BindAndValidate(c, req.Interface()); err != nil {
			return err
		}

		res := fVal.Call([]reflect.Value{reflect.ValueOf(c), req.Elem()})
		if !res[errorIndex].IsNil() {
			err, ok := res[errorIndex].Interface().(error)
			if !ok {
				return fmt.Errorf("could not cast error index: %+v", res[errorIndex].Interface())
			}
			if err != nil {
				return err
			}
		}

		resp := c.Response()
		if numOut == 2 || !resp.Committed {
			data := res[0].Interface()
			resp := &Response{
				Status:  http.StatusOK,
				Success: true,
				Data:    data,
			}
			if v, ok := data.(*Response); ok {
				resp = v
			}
			return c.JSON(resp.Status, resp)
		}

		if numOut == 1 && !resp.Committed {
			resp.Header().Del(echo.HeaderContentType)
			return c.NoContent(http.StatusNoContent)
		}

		return nil
	}

	return handler, nil
}
