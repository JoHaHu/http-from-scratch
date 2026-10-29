package mux1

import (
	"errors"
	"maps"
	"net/http"
	"strings"
	"unicode/utf8"
)

type PathMux struct {
	routes matcher
}

func NewMux() *PathMux {
	return &PathMux{
		routes: matcher{
			root: NewTrie(),
		},
	}

}

func (m *PathMux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler := m.routes.Match(request.URL.EscapedPath())
	if handler == nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	handler.ServeHTTP(writer, request)
}

func (m *PathMux) Register(route string, f func(w http.ResponseWriter, r *http.Request)) {
	m.routes.Add(route, http.HandlerFunc(f))
}
func (m *PathMux) RegisterHandler(route string, h http.Handler) {
	m.routes.Add(route, h)
}

type MethodMux struct {
	methods map[string]http.Handler
}

func NewMethodMux() *MethodMux {
	return &MethodMux{
		methods: make(map[string]http.Handler),
	}
}
func (m *MethodMux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler := m.methods[request.Method]
	if handler == nil {
		http.Error(writer, "404 page not found", http.StatusNotFound)
	}
	handler.ServeHTTP(writer, request)
}

func (m *MethodMux) Get(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["GET"] = http.HandlerFunc(f)
	return m
}
func (m *MethodMux) Post(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["POST"] = http.HandlerFunc(f)
	return m
}

func (m *MethodMux) Put(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["PUT"] = http.HandlerFunc(f)
	return m
}
func (m *MethodMux) Patch(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["PATCH"] = http.HandlerFunc(f)
	return m
}
func (m *MethodMux) Options(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["OPTIONS"] = http.HandlerFunc(f)
	return m
}
func (m *MethodMux) Head(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["HEAD"] = http.HandlerFunc(f)
	return m
}
func (m *MethodMux) Connect(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["CONNECT"] = http.HandlerFunc(f)
	return m
}
func (m *MethodMux) Trace(f func(w http.ResponseWriter, r *http.Request)) *MethodMux {
	m.methods["TRACE"] = http.HandlerFunc(f)
	return m
}

type matcher struct {
	root *trieNode
}

type trieNode struct {
	children map[string]*trieNode
	value    http.Handler
}

func NewTrie() *trieNode {
	return &trieNode{
		children: make(map[string]*trieNode),
	}
}

func (m matcher) Add(path string, handler http.Handler) error {

	var node = m.root
	currentPath := path

	for {
		prefixes := maps.Keys(node.children)
		foundPrefix := false
		prefix := ""
		for prefix = range prefixes {
			if strings.HasPrefix(currentPath, prefix) {
				currentPath, foundPrefix = strings.CutPrefix(path, prefix)
				node = node.children[prefix]
				break
			}
		}
		if foundPrefix {
			if len(currentPath) != 0 {
				continue
			}
			if len(prefix) == 0 {
				if node.value == nil {
					node.value = handler
					return nil
				}
				return errors.New("handler already in use")
			}
		} else {
			for prefix = range prefixes {
				cp, found := commonPrefixRune(prefix, currentPath)
				if found {
					oldNode := node.children[prefix]
					oldPrefix, _ := strings.CutPrefix(prefix, cp)
					delete(node.children, prefix)

					newPrefix, _ := strings.CutPrefix(currentPath, cp)
					newNode := NewTrie()

					node.children[cp] = newNode
					node = newNode

					if len(oldPrefix) == 0 {
						node.value = oldNode.value
					} else {
						node.children[oldPrefix] = oldNode
					}

					if len(newPrefix) == 0 {
						node.value = handler
					} else {
						node.children[newPrefix] = &trieNode{
							children: make(map[string]*trieNode),
							value:    handler,
						}
					}
					return nil
				}
			}
			newNode := &trieNode{
				children: make(map[string]*trieNode),
				value:    handler,
			}
			node.children[currentPath] = newNode

		}
		return nil
	}
}

func (m matcher) Match(path string) http.Handler {

	var node = m.root
	currentPath := path
	for {
		prefixes := maps.Keys(node.children)
		foundPrefix := false
		prefix := ""
		for prefix = range prefixes {

			if strings.HasPrefix(currentPath, prefix) {
				currentPath, foundPrefix = strings.CutPrefix(currentPath, prefix)
				node = node.children[prefix]
				break
			}
		}
		if foundPrefix {
			if len(currentPath) != 0 {
				continue
			}
			return node.value
		}
		return nil
	}
}

func commonPrefixRune(a, b string) (string, bool) {
	var i int
	for {
		r1, size1 := utf8.DecodeRuneInString(a[i:])
		r2, size2 := utf8.DecodeRuneInString(b[i:])

		if size1 == 0 || size2 == 0 || r1 != r2 {
			break
		}
		i += size1
	}
	return a[:i], i > 0
}
