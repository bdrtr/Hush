package hush

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/go-playground/validator/v10"
	"github.com/valyala/fasthttp"
)

// BindBody reads JSON from fasthttp request body and parses into type T.
func BindBody[T any](c *Context) (*T, error) {
	var obj T
	
	contentType := c.Ctx.Request.Header.Peek("Content-Type")
	if !bytes.HasPrefix(contentType, []byte("application/json")) {
		err := fmt.Errorf("unsupported content type: expected application/json")
		c.Ctx.Error(err.Error(), fasthttp.StatusUnsupportedMediaType)
		c.Abort()
		return nil, err
	}

	body := c.Ctx.Request.Body()
	if len(body) == 0 {
		err := fmt.Errorf("request body cannot be empty")
		c.Ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		c.Abort()
		return nil, err
	}
	
	if err := sonic.Unmarshal(body, &obj); err != nil {
		c.Ctx.Error("Invalid JSON body", fasthttp.StatusBadRequest)
		c.Abort()
		return &obj, err
	}
	
	if err := validateStruct(&obj); err != nil {
		return nil, err
	}
	
	return &obj, nil
}

// BindQuery parses URL query parameters into type T based on struct tags.
func BindQuery[T any](c *Context) (*T, error) {
	var obj T
	
	val := reflect.ValueOf(&obj).Elem()
	typ := val.Type()
	
	args := c.Ctx.QueryArgs()
	
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("query")
		if tag == "" {
			continue
		}
		
		queryVal := args.Peek(tag)
		if len(queryVal) > 0 {
			if val.Field(i).Kind() == reflect.String {
				val.Field(i).SetString(string(queryVal))
			}
			// Extensions for Int, Bool etc. can be added here
		}
	}
	
	if err := validateStruct(&obj); err != nil {
		return nil, err
	}
	
	return &obj, nil
}

var validate = validator.New()

// validateStruct uses go-playground/validator/v10 to enforce advanced validation rules.
func validateStruct(obj interface{}) error {
	return validate.Struct(obj)
}
