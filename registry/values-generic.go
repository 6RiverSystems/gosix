// Copyright (c) 2023 6 River Systems
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
	"reflect"
)

type genericKey[T any] struct {
	address   string
	valueType reflect.Type
}

var _ TypedKey[int] = (*genericKey[int])(nil)

func (p *genericKey[T]) Address() string         { return p.address }
func (p *genericKey[T]) ValueType() reflect.Type { return p.valueType }

func (s *genericKey[T]) Value(vv Values) (T, bool) {
	v, ok := vv.Value(s)
	if !ok {
		var zero T
		return zero, false
	}
	vs, ok := v.(T)
	return vs, ok
}

func ValueAt[T any](address string) TypedKey[T] {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return &genericKey[T]{address, t}
}
