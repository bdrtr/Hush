package hush

import "reflect"

// Route represents a registered route and its metadata for OpenAPI.
// WARNING: Route builder methods (WithTags, WithSummary, etc.) are NOT thread-safe.
// They should only be called sequentially during server initialization, not concurrently.
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
