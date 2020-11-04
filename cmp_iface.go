package di

import (
	"reflect"
)

type interfaceCompiler struct {
	node *node
}

func newInterfaceCompiler(n *node) *interfaceCompiler {
	return &interfaceCompiler{node: n}
}

func (i interfaceCompiler) params(s schema) (params []*node, err error) {
	return append(params, i.node), nil
}

func (i interfaceCompiler) compile(_ []reflect.Value, s schema) (reflect.Value, error) {
	return i.node.Value(s)
}
