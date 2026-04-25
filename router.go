package hush

import (
	"log"
	"strings"

	"github.com/cespare/xxhash/v2"
)

// Router represents the Radix Tree based router.
type Router struct {
	routes      map[string]*node            // HTTP Method -> Radix Tree Root
	staticCache map[string]map[uint64]*node // HTTP Method -> Hash -> Node
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
		routes:      make(map[string]*node),
		staticCache: make(map[string]map[uint64]*node),
	}
}

// insert adds a new route to the radix tree.
func (r *Router) insert(method, path string, handlers []HandlerFunc) {
	isStatic := !strings.Contains(path, ":") && !strings.Contains(path, "*")

	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &node{}
		r.staticCache[method] = make(map[uint64]*node)
	}
	
	parts := parsePath(path)
	root := r.routes[method]
	
	for _, part := range parts {
		isWild := strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*")
		var child *node
		for _, c := range root.children {
			if c.part == part {
				child = c
				break
			}
			if isWild && c.isWild {
				panic("route conflict: cannot register '" + part + "' because '" + c.part + "' already exists at this path segment")
			}
		}
		
		if child == nil {
			child = &node{
				part:   part,
				isWild: isWild,
			}
			if isWild {
				root.children = append(root.children, child)
			} else {
				root.children = append([]*node{child}, root.children...)
			}
		}
		root = child
	}
	root.path = path
	root.handlers = handlers

	// Cache pure static routes for O(1) access
	if isStatic {
		hash := xxhash.Sum64String(path)
		r.staticCache[method][hash] = root
	}
}

// get finds the route and populates parameters directly into Context (zero alloc)
func (r *Router) get(method, path string, c *Context) *node {
	// O(1) Fast-Path for purely static routes
	if cache, ok := r.staticCache[method]; ok {
		hash := xxhash.Sum64String(path)
		if n, found := cache[hash]; found {
			if n.path == path {
				return n
			}
			log.Printf("[HUSH-WARN] Hash collision detected in static route cache: '%s' vs '%s'. Falling back to tree traversal.", n.path, path)
		}
	}

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
		if n.handlers != nil {
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
			initialParamCount := c.paramCount
			if child.isWild {
				if strings.HasPrefix(child.part, "*") {
					// Catch-all: consume the rest of the path entirely
					c.addParam(child.part[1:], path)
					if child.handlers != nil {
						return child
					}
					return nil
				}
				// Normal parameter: avoid allocations
				c.addParam(child.part[1:], part)
			}
			result := r.search(child, rest, c)
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
