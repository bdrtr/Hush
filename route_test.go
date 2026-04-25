package hush

import (
	"reflect"
	"testing"
)

func TestRoute_WithTagsDedup(t *testing.T) {
	r := &Route{}
	r.WithTags("users", "admin").WithTags("users", "billing")

	if len(r.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(r.Tags))
	}

	expected := map[string]bool{"users": true, "admin": true, "billing": true}
	for _, tag := range r.Tags {
		if !expected[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
	}
}

func TestRoute_WithBodyInterfacePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic when passing an interface to WithBody")
		}
	}()

	r := &Route{}
	// This should panic because any interface type evaluates to interface Kind
	WithBody[error](r)
}

func TestRoute_WithResponseInterfacePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic when passing an interface to WithResponse")
		}
	}()

	r := &Route{}
	WithResponse[error](r, 200, "Success")
}

func TestRoute_Getters(t *testing.T) {
	r := &Route{
		method: "POST",
		path:   "/users",
	}

	if r.Method() != "POST" {
		t.Errorf("Expected method POST, got %s", r.Method())
	}

	if r.Path() != "/users" {
		t.Errorf("Expected path /users, got %s", r.Path())
	}
}
