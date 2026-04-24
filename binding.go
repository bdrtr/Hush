package hush

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// BindBody reads JSON from request body and parses into type T.
func BindBody[T any](c *Context) (*T, error) {
	var obj T
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&obj); err != nil {
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
	
	queries := c.Request.URL.Query()
	
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("query")
		if tag == "" {
			continue
		}
		
		if queryVal := queries.Get(tag); queryVal != "" {
			if val.Field(i).Kind() == reflect.String {
				val.Field(i).SetString(queryVal)
			}
			// For Phase 1 we only support string query params to keep it simple.
			// Int, Bool etc parsing would go here.
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
				// Other rules like min, max, email can be added here
			}
		}
	}
	return nil
}
