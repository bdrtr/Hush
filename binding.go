package hush

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
)

// BindBody reads JSON from fasthttp request body and parses into type T.
func BindBody[T any](c *Context) (*T, error) {
	var obj T
	body := c.Ctx.PostBody()
	
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
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
