package di

import (
	"fmt"
	"reflect"

	"github.com/emicklei/dot"
)

// schema is a dependency injection schema.
type schema interface {
	// find finds reflect.Type with matching Tags.
	find(t reflect.Type, tags Tags) (*node, error)
	// register cleanup
	cleanup(cleanup func())
}

// schema is a dependency injection schema.
type defaultSchema struct {
	nodes    map[reflect.Type][]*node
	cleanups []func()
	passed   map[*node]int
}

func (s *defaultSchema) cleanup(cleanup func()) {
	s.cleanups = append(s.cleanups, cleanup)
}

// newDefaultSchema creates new dependency injection schema.
func newDefaultSchema() *defaultSchema {
	return &defaultSchema{
		nodes: map[reflect.Type][]*node{},
	}
}

// register registers reflect.Type provide function with optional Tags. Also, its registers
// type []<type> for group.
func (s *defaultSchema) register(n *node) {
	if _, ok := s.nodes[n.rt]; !ok {
		s.nodes[n.rt] = []*node{n}
		return
	}
	s.nodes[n.rt] = append(s.nodes[n.rt], n)
}

// find finds provideFunc by its reflect.Type and Tags.
func (s *defaultSchema) find(t reflect.Type, tags Tags) (*node, error) {
	nodes, ok := s.nodes[t]
	if !ok && t.Kind() != reflect.Slice && !isInjectable(t) {
		return nil, fmt.Errorf("type %s%s not exists in the container", t, tags)
	}
	// type found
	if ok {
		matched := matchTags(nodes, tags)
		if len(matched) == 0 {
			return nil, fmt.Errorf("%s%s not exists", t, tags)
		}
		if len(matched) > 1 {
			return nil, fmt.Errorf("multiple definitions of %s%s, maybe you need to use group type: []%s%s", t, tags, t, tags)
		}
		return matched[0], nil
	}
	if !ok && isInjectable(t) {
		// constructor result with di.Inject - only addressable pointers
		// anonymous parameters with di.Inject - only struct
		if t.Kind() == reflect.Ptr {
			return nil, fmt.Errorf("inject %s%s fields not supported, use %s%s", t, tags, t.Elem(), tags)
		}
		node := &node{
			rv:       &reflect.Value{},
			rt:       t,
			compiler: newTypeCompiler(t),
		}
		// save node for future use
		s.nodes[t] = append(s.nodes[t], node)
		return node, nil
	}
	return s.group(t, tags)
}

func (s *defaultSchema) group(t reflect.Type, tags Tags) (*node, error) {
	group, ok := s.nodes[t.Elem()]
	if !ok {
		return nil, fmt.Errorf("%s%s not exists", t, tags)
	}
	matched := matchTags(group, tags)
	if len(matched) == 0 {
		return nil, fmt.Errorf("%s%s not exists", t, tags)
	}
	node := &node{
		rv:       &reflect.Value{},
		rt:       t,
		tags:     tags,
		compiler: newGroupCompiler(t, matched),
	}
	return node, nil
}

func (s *defaultSchema) bypass(n *node) error {
	if s.passed == nil {
		s.passed = map[*node]int{}
		for _, group := range s.nodes {
			for _, node := range group {
				if err := visit(s, node, s.passed); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(s, n, s.passed); err != nil {
		return err
	}
	return nil
}

func (s *defaultSchema) renderDot() (*dot.Graph, error) {
	root := dot.NewGraph()
	for _, group := range s.nodes {
		for _, n := range group {
			if err := s.renderNode(root, n); err != nil {
				return nil, err
			}
		}
	}
	return root, nil
}

func (s *defaultSchema) renderNode(root *dot.Graph, n *node) error {
	rnode := root.Node(n.String())
	var all []*node
	params, err := n.params(s)
	if err != nil {
		return err
	}
	all = append(all, params...)
	for _, field := range n.fields() {
		fnode, err := s.find(field.rt, field.tags)
		if err != nil {
			return err
		}
		all = append(all, fnode)
	}
	for _, p := range all {
		if err := s.renderNode(root, p); err != nil {
			return err
		}
		if len(root.Node(p.String()).EdgesTo(rnode)) == 0 {
			root.Node(p.String()).Edge(rnode)
		}
	}
	return nil
}
