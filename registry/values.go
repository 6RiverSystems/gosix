// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package registry

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
)

type Key interface {
	Address() string
	ValueType() reflect.Type
}

type TypedKey[T any] interface {
	Key
	Value(Values) (value T, ok bool)
}

type ValueSource interface {
	Value(Values) any
	ValueType() reflect.Type
}

// TODO: TypedValueSource

type exactValue struct{ value any }

func (v exactValue) Value(Values) any {
	return v.value
}

func (v exactValue) ValueType() reflect.Type {
	return reflect.ValueOf(v.value).Type()
}

func ConstantValue(v any) ValueSource {
	return exactValue{v}
}

type provider[T any] struct {
	f func(Values) T
	t reflect.Type
}

func (p *provider[T]) Value(v Values) any {
	return p.f(v)
}

func (p *provider[T]) ValueType() reflect.Type {
	return p.t
}

func Provider[T any](f func(Values) T) *provider[T] {
	fv := reflect.ValueOf(f)
	fvt := fv.Type()
	if fvt.Kind() != reflect.Func {
		panic(fmt.Errorf("Provider called with a non-function argument"))
	}
	if fvt.NumIn() != 1 || fvt.IsVariadic() || fvt.In(0) != valuesType {
		panic(fmt.Errorf("Provider called with function with wrong arguments"))
	}
	if fvt.NumOut() != 1 {
		panic(fmt.Errorf("Provider called with function with wrong number of outputs"))
	}
	return &provider[T]{
		f: f,
		t: fvt.Out(0),
	}
}

func EnvProvider(name string) *provider[string] {
	return Provider(func(Values) string {
		return os.Getenv(name)
	})
}

type alias struct{ Key }

var _ ValueSource = alias{}

func (a alias) Value(vs Values) any {
	if v, ok := vs.Value(a.Key); ok {
		return v
	}
	panic(fmt.Errorf("Unable to resolve alias to %s (%v)", a.Key.Address(), a.Key))
}
func Alias(k Key) ValueSource { return alias{k} }

type Values interface {
	Path() string
	ValueSource(Key) (ValueSource, bool)
	Value(Key) (any, bool)
}

type adapter[T, U any] struct {
	ValueSource
	f func(T) U
}

func (a adapter[T, U]) Value(vs Values) any {
	anyT := a.ValueSource.Value(vs)
	if t, ok := anyT.(T); ok {
		return a.f(t)
	} else {
		panic(fmt.Errorf("Unable to adapt: value is %T not %T", anyT, t))
	}
}

func (adapter[T, U]) ValueType() reflect.Type {
	return reflect.TypeOf((*U)(nil)).Elem()
}

func Adapter[T, U any](vs ValueSource, f func(T) U) ValueSource {
	return adapter[T, U]{vs, f}
}

var valuesType = reflect.TypeOf((*Values)(nil)).Elem()

type MutableValues interface {
	Values
	Bind(Key, ValueSource) bool
	// TODO: want this in a helper method so it doesn't need to be re-implemented
	MustBind(Key, ValueSource)
}

type rootContainer map[Key]ValueSource

var _ MutableValues = rootContainer(nil)

func (rootContainer) Path() string {
	return "/"
}

func (c rootContainer) Value(k Key) (any, bool) {
	vs, ok := c[k]
	if !ok {
		return nil, false
	}
	return vs.Value(c), true
}

func (c rootContainer) ValueSource(k Key) (ValueSource, bool) {
	vs, ok := c[k]
	return vs, ok
}

func (c rootContainer) Bind(k Key, s ValueSource) bool {
	t := s.ValueType()
	if !t.AssignableTo(k.ValueType()) {
		return false
	}
	c[k] = s
	return true
}

func (c rootContainer) MustBind(k Key, s ValueSource) {
	if ok := c.Bind(k, s); !ok {
		panic(fmt.Errorf("unable to bind %v to %v, type mismatch?", k, s))
	}
}

func NewValues() MutableValues {
	return make(rootContainer)
}

type childContainer struct {
	parent Values
	name   string
	values rootContainer
}

var _ MutableValues = (*childContainer)(nil)

func (c *childContainer) Path() string {
	if c.parent != nil {
		return c.parent.Path() + c.name + "/"
	}
	return "/" + c.name + "/"
}

func (c *childContainer) ValueSource(k Key) (ValueSource, bool) {
	if vs, ok := c.values[k]; ok {
		return vs, true
	} else if c.parent != nil {
		return c.parent.ValueSource(k)
	}
	return nil, false
}

func (c *childContainer) Value(k Key) (any, bool) {
	if vs, ok := c.ValueSource(k); !ok {
		return nil, false
	} else {
		// evaluate the provider in the context of this _leaf_ container, not in the context of where it was bound!
		return vs.Value(c), true
	}
}

func (c *childContainer) Bind(k Key, s ValueSource) bool {
	return c.values.Bind(k, s)
}

func (c *childContainer) MustBind(k Key, s ValueSource) {
	c.values.MustBind(k, s)
}

func ChildValues(parent Values, name string) MutableValues {
	return &childContainer{parent, name, make(rootContainer)}
}

type cachedContainer struct {
	Values
	mu    sync.Mutex
	cache map[Key]any
}

func (c *cachedContainer) Path() string {
	p := c.Values.Path()
	// assumes p endswith /
	return p[0:len(p)-1] + "(c)/"
}

func (c *cachedContainer) Value(k Key) (any, bool) {
	if v, ok := c.get(k); ok {
		return v, true
	} else if vs, ok := c.Values.ValueSource(k); ok {
		v := vs.Value(c)
		c.put(k, v)
		return v, true
	}
	return nil, false
}

func (c *cachedContainer) get(k Key) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.cache[k]
	return v, ok
}

func (c *cachedContainer) put(k Key, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[k] = v
}

func CachedValues(vs Values) Values {
	if cc, ok := vs.(*cachedContainer); ok {
		return cc
	}
	return &cachedContainer{Values: vs, cache: map[Key]any{}}
}

type contextKey struct{ name string }

var contextValues = &contextKey{"registry.Values"}

func ContextValues(ctx context.Context) Values {
	return ctx.Value(contextValues).(Values)
}

func WithValues(ctx context.Context, vs Values) context.Context {
	return context.WithValue(ctx, contextValues, vs)
}

func WithChildValues(ctx context.Context, name string) (context.Context, MutableValues) {
	// child container can cope with a null parent
	child := ChildValues(ContextValues(ctx), name)
	return WithValues(ctx, child), child
}

func WithCachedChildValues(ctx context.Context, name string) (context.Context, Values) {
	// child container can cope with a null parent
	child := CachedValues(ChildValues(ContextValues(ctx), name))
	return WithValues(ctx, child), child
}
