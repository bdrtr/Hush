package hush

import (
	"fmt"
	"reflect"
	"strings"

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

// validateStruct uses reflection to enforce basic validation tags like "required".
func validateStruct(obj interface{}) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return nil
	}
	
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("validate")
		
		if tag != "" {
			rules := strings.Split(tag, ",")
			fieldVal := val.Field(i)
			
			for _, rule := range rules {
				if rule == "required" {
					if fieldVal.IsZero() {
						return fmt.Errorf("field %s is required", field.Name)
					}
				}
			}
		}
	}
	return nil
}
