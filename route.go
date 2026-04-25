package hush

import "reflect"

// Route represents a registered route and its metadata for OpenAPI.
// WARNING: Route builder methods (WithTags, WithSummary, etc.) are NOT thread-safe.
// They should only be called sequentially during server initialization, not concurrently.
type Route struct {
	method       string
	path         string
	Summary      string
	Tags         []string
	RequestBody  reflect.Type
	ResponseBody reflect.Type
	QueryParams  reflect.Type
	HeaderParams reflect.Type
}

// Method returns the HTTP method of the route.
func (r *Route) Method() string {
	return r.method
}

// Path returns the path of the route.
func (r *Route) Path() string {
	return r.path
}

// WithSummary adds a summary to the route for OpenAPI.
// Note: This method is not thread-safe.
func (r *Route) WithSummary(summary string) *Route {
	r.Summary = summary
	return r
}

// WithTags adds tags to the route for OpenAPI.
// Note: This method is not thread-safe.
func (r *Route) WithTags(tags ...string) *Route {
	seen := make(map[string]bool, len(r.Tags))
	for _, t := range r.Tags {
		seen[t] = true
	}
	for _, t := range tags {
		if !seen[t] {
			r.Tags = append(r.Tags, t)
			seen[t] = true
		}
	}
	return r
}

// WithBody documents the request body type for OpenAPI.
// Note: Due to Go generics limitations, this must be called as a wrapper:
//
//	hush.WithBody[MyReq](route.WithSummary("x"))
func WithBody[T any](r *Route) *Route {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Interface {
		panic("hush: WithBody does not support interface types, use a concrete struct")
	}
	r.RequestBody = t
	return r
}

// WithResponse documents the response body type for OpenAPI.
// Note: Due to Go generics limitations, this must be called as a wrapper.
func WithResponse[T any](r *Route) *Route {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Interface {
		panic("hush: WithResponse does not support interface types, use a concrete struct")
	}
	r.ResponseBody = t
	return r
}

// WithQuery documents the query parameters type for OpenAPI.
// Note: Due to Go generics limitations, this must be called as a wrapper.
func WithQuery[T any](r *Route) *Route {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Interface {
		panic("hush: WithQuery does not support interface types, use a concrete struct")
	}
	r.QueryParams = t
	return r
}

// WithHeader documents the header parameters type for OpenAPI.
// Note: Due to Go generics limitations, this must be called as a wrapper.
func WithHeader[T any](r *Route) *Route {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Interface {
		panic("hush: WithHeader does not support interface types, use a concrete struct")
	}
	r.HeaderParams = t
	return r
}
