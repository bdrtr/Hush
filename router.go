package hush

import (
	"bytes"
	"strings"
	"unsafe"
)

// Router represents the Radix Tree based router.
type Router struct {
	routes map[string]*node // HTTP Method -> Radix Tree Root
}

type node struct {
	path     string
	part     string
	children []*node
	isWild   bool
	handlers []HandlerFunc
}

func newRouter() *Router {
	return &Router{
		routes: make(map[string]*node),
	}
}

// insert adds a new route to the radix tree.
func (r *Router) insert(method, path string, handlers []HandlerFunc) {
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &node{}
	}
	
	parts := parsePath(path)
	root := r.routes[method]
	
	for _, part := range parts {
		child := root.matchChild(part)
		if child == nil {
			child = &node{
				part:   part,
				isWild: strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*"),
			}
			root.children = append(root.children, child)
		}
		root = child
	}
	root.path = path
	root.handlers = handlers
}

// get finds the route and populates parameters directly into Context (zero alloc)
func (r *Router) get(method, path string, c *Context) *node {
	root, ok := r.routes[method]
	if !ok {
		return nil
	}
	
	return r.search(root, path, c)
}

func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// getByte finds the route using zero-allocation byte traversal
func (r *Router) getByte(method string, path []byte, c *Context) *node {
	root, ok := r.routes[method]
	if !ok {
		return nil
	}
	
	return r.searchByte(root, path, c)
}

// searchByte is the highly optimized radix tree search using bytes and unsafe strings
func (r *Router) searchByte(n *node, path []byte, c *Context) *node {
	// Strip leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if len(path) == 0 {
		if n.path != "" {
			return n
		}
		return nil
	}

	var part, rest []byte
	idx := bytes.IndexByte(path, '/')
	if idx == -1 {
		part = path
		rest = nil
	} else {
		part = path[:idx]
		rest = path[idx+1:]
	}

	partStr := unsafeString(part)

	for _, child := range n.children {
		if child.part == partStr || child.isWild {
			initialParamCount := c.paramCount
			if child.isWild {
				if strings.HasPrefix(child.part, "*") {
					// Catch-all: consume the rest of the path entirely
					c.addParam(child.part[1:], unsafeString(path))
					if child.handlers != nil {
						return child
					}
					return nil
				}
				// Normal parameter: zero allocations
				c.addParam(child.part[1:], partStr)
			}
			result := r.searchByte(child, rest, c)
			if result != nil {
				return result
			}
			// Backtrack: restore param count if search down this path failed
			c.paramCount = initialParamCount
		}
	}
	return nil
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part {
			return child
		}
	}
	return nil
}

func parsePath(path string) []string {
	parts := strings.Split(path, "/")
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
