package registry

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Types(t *testing.T) {

	var _gen = 0
	gen := func(Values) int {
		_gen++
		return _gen
	}

	p := Provider(gen)

	assert.Equal(t, reflect.TypeOf(int(0)), p.ValueType())
	v1 := p.Value(nil)
	v2 := p.Value(nil)
	assert.Equal(t, 1, v1)
	assert.Equal(t, 2, v2)
	assert.IsType(t, int(0), v1)
	assert.IsType(t, int(0), v2)
}

func TestKey_Duplicates(t *testing.T) {
	tests := []struct {
		factory   func(string) Key
		generator func() []interface{}
	}{
		{
			StringAt,
			func() []interface{} { return []interface{}{"1", "2"} },
		},
		{
			Int64At,
			func() []interface{} { return []interface{}{int64(0), int64(1)} },
		},
		{
			func(s string) Key { return PointerAt(s, (*struct{ placeholder bool })(nil)) },
			func() []interface{} {
				return []interface{}{
					&struct{ placeholder bool }{false},
					&struct{ placeholder bool }{true},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", shortName(tt.factory)), func(t *testing.T) {
			var keys []Key
			var values []interface{}
			c := NewValues()
			for i, v := range tt.generator() {
				k := tt.factory(t.Name())
				assert.True(t, c.Bind(k, ConstantValue(v)))
				keys = append(keys, k)
				values = append(values, v)
				if i > 0 {
					assert.NotEqual(t, values[i-1], v)
				}
			}
			for i, k := range keys {
				v, ok := c.Value(k)
				if assert.True(t, ok) {
					assert.Equal(t, values[i], v)
				}
			}
		})
	}
}

func shortName(f interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if p := strings.LastIndex(name, "."); p >= 0 {
		name = name[p+1:]
	}
	return name
}

func TestLayeredCache(t *testing.T) {
	k1 := Int64At("k1")
	k2 := Int64At("k2")
	k3 := Int64At("k3")

	k1p1 := Provider(func(vs Values) int64 {
		v, ok := vs.Value(k1)
		require.True(t, ok)
		return v.(int64) + 1
	})
	k2a := Alias(k2)

	r := NewValues()
	c1 := ChildValues(r, "c1")
	c2 := ChildValues(r, "c2")
	cc1 := CachedValues(c1)
	cc2 := CachedValues(c2)

	assert.Equal(t, "/", r.Path())
	assert.Equal(t, "/c1/", c1.Path())
	assert.Equal(t, "/c2/", c2.Path())
	assert.Equal(t, "/c1(c)/", cc1.Path())
	assert.Equal(t, "/c2(c)/", cc2.Path())

	// at the root, k2 is a dynamic provider for k1+1
	assert.True(t, r.Bind(k2, k1p1))
	// in child i, k1 has value 10i
	assert.True(t, c1.Bind(k1, ConstantValue(int64(10))))
	assert.True(t, c2.Bind(k1, ConstantValue(int64(20))))

	// should be able to resolve k2 as 11, 21 in c1, c2
	assert.Equal(t, int64(11), MustValue(t, c1, k2))
	assert.Equal(t, int64(21), MustValue(t, c2, k2))

	// fetching through the cache and then modifying the values should change the
	// direct loader but not the cached value
	assert.Equal(t, int64(11), MustValue(t, cc1, k2))
	assert.Equal(t, int64(21), MustValue(t, cc2, k2))
	assert.True(t, c1.Bind(k1, ConstantValue(int64(11))))
	assert.True(t, c2.Bind(k1, ConstantValue(int64(21))))
	assert.Equal(t, int64(11), MustValue(t, cc1, k2))
	assert.Equal(t, int64(21), MustValue(t, cc2, k2))
	assert.Equal(t, int64(12), MustValue(t, c1, k2))
	assert.Equal(t, int64(22), MustValue(t, c2, k2))

	// the alias should see the cached value when resolved through the cache, even
	// when bound after the cache was loaded
	assert.True(t, r.Bind(k3, k2a))
	assert.Equal(t, int64(12), MustValue(t, c1, k3))
	assert.Equal(t, int64(22), MustValue(t, c2, k3))
	assert.Equal(t, int64(11), MustValue(t, cc1, k3))
	assert.Equal(t, int64(21), MustValue(t, cc2, k3))
}

func MustValue(t testing.TB, vs Values, k Key) interface{} {
	t.Helper()
	v, ok := vs.Value(k)
	require.True(t, ok)
	return v
}
