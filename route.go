package hush

import "reflect"

// Route represents a registered route and its metadata for OpenAPI
type Route struct {
	Method       string
	Path         string
	Summary      string
	Tags         []string
	RequestBody  reflect.Type
	ResponseBody reflect.Type
	QueryParams  reflect.Type
	HeaderParams reflect.Type
}

// WithSummary adds a summary to the route for OpenAPI
func (r *Route) WithSummary(summary string) *Route {
	r.Summary = summary
	return r
}

// WithTags adds tags to the route for OpenAPI
func (r *Route) WithTags(tags ...string) *Route {
	r.Tags = append(r.Tags, tags...)
	return r
}

// WithBody documents the request body type for OpenAPI
func WithBody[T any](r *Route) *Route {
	r.RequestBody = reflect.TypeOf((*T)(nil)).Elem()
	return r
}

// WithResponse documents the response body type for OpenAPI
func WithResponse[T any](r *Route) *Route {
	r.ResponseBody = reflect.TypeOf((*T)(nil)).Elem()
	return r
}

// WithQuery documents the query parameters type for OpenAPI
func WithQuery[T any](r *Route) *Route {
	r.QueryParams = reflect.TypeOf((*T)(nil)).Elem()
	return r
}

// WithHeader documents the header parameters type for OpenAPI
func WithHeader[T any](r *Route) *Route {
	r.HeaderParams = reflect.TypeOf((*T)(nil)).Elem()
	return r
}
