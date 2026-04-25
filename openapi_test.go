package hush

import (
	"reflect"
	"testing"
	"time"
)

type RecursiveNode struct {
	Value int
	Next  *RecursiveNode
}

func TestOpenAPI_BuildSchemaCycleDetection(t *testing.T) {
	// If the cycle detection is broken, this will stack overflow
	schema := buildSchema(reflect.TypeOf(RecursiveNode{}))
	
	if schema["type"] != "object" {
		t.Errorf("Expected root type object, got %v", schema["type"])
	}
	
	props := schema["properties"].(map[string]interface{})
	if props["Value"].(map[string]interface{})["type"] != "integer" {
		t.Errorf("Expected Value type integer")
	}
	
	// The cycle breaker returns an empty object schema for the recursive reference
	nextProp := props["Next"].(map[string]interface{})
	if nextProp["type"] != "object" {
		t.Errorf("Expected Next type object due to cycle break, got %v", nextProp["type"])
	}
}

type TimeMapStruct struct {
	CreatedAt time.Time
	Metadata  map[string]string
}

func TestOpenAPI_SpecialTypes(t *testing.T) {
	schema := buildSchema(reflect.TypeOf(TimeMapStruct{}))
	props := schema["properties"].(map[string]interface{})
	
	createdAt := props["CreatedAt"].(map[string]interface{})
	if createdAt["type"] != "string" {
		t.Errorf("Expected time.Time to map to type string, got %v", createdAt["type"])
	}
	if createdAt["format"] != "date-time" {
		t.Errorf("Expected time.Time to map to format date-time, got %v", createdAt["format"])
	}
	
	metadata := props["Metadata"].(map[string]interface{})
	if metadata["type"] != "object" {
		t.Errorf("Expected map to map to type object, got %v", metadata["type"])
	}
	if metadata["additionalProperties"] == nil {
		t.Errorf("Expected map to have additionalProperties")
	}
}

func TestOpenAPI_EmptyTagsAndSummary(t *testing.T) {
	e := New()
	r := e.GET("/test", func(c *Context) {})
	// Intentionally don't set Tags or Summary
	
	doc := e.GenerateOpenAPI("Test", "1.0")
	paths := doc["paths"].(map[string]interface{})
	testPath := paths["/test"].(map[string]interface{})
	getOp := testPath["get"].(map[string]interface{})
	
	if _, exists := getOp["summary"]; exists {
		t.Errorf("Expected empty summary to be omitted, but it was included")
	}
	
	if _, exists := getOp["tags"]; exists {
		t.Errorf("Expected empty tags to be omitted, but it was included")
	}
	
	// Now test setting them
	r.WithSummary("Hello").WithTags("test")
	doc2 := e.GenerateOpenAPI("Test", "1.0")
	getOp2 := doc2["paths"].(map[string]interface{})["/test"].(map[string]interface{})["get"].(map[string]interface{})
	
	if getOp2["summary"] != "Hello" {
		t.Errorf("Expected summary Hello, got %v", getOp2["summary"])
	}
}
