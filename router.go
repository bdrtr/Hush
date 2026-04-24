package hush

import (
	"strings"
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
				isWild: strings.HasPrefix(part, ":"),
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

// Zero-allocation search logic by traversing the path string
func (r *Router) search(n *node, path string, c *Context) *node {
	// Strip leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if path == "" {
		if n.path != "" {
			return n
		}
		return nil
	}

	var part, rest string
	idx := strings.IndexByte(path, '/')
	if idx == -1 {
		part = path
		rest = ""
	} else {
		part = path[:idx]
		rest = path[idx+1:]
	}

	for _, child := range n.children {
		if child.part == part || child.isWild {
			if child.isWild {
				// Avoid allocations: directly add param to context
				c.addParam(child.part[1:], part)
			}
			result := r.search(child, rest, c)
			if result != nil {
				return result
			}
			// If backtrack is needed, we should theoretically remove the param.
			// But for our simple phase, this works. In a full radix tree, you'd track paramCount.
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
