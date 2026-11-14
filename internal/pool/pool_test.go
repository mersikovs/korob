package pool_test

import (
	"testing"

	"github.com/mersikovs/korob.git/internal/pool"
)

var _ pool.Resettable = (*testObj)(nil)

type testObj struct {
	value  int
	data   []byte
	resets int
}

func (t *testObj) Reset() {
	t.resets++
	t.value = 0
	t.data = t.data[:0]
}

func TestGet_FreshObject(t *testing.T) {
	p := pool.New(func() *testObj {
		return &testObj{value: -1}
	})

	obj := p.Get()
	if obj == nil {
		t.Fatal("Get returned nil")
	}
	if obj.value != -1 {
		t.Fatalf("want value=-1 from factory, got %d", obj.value)
	}
	if obj.resets != 0 {
		t.Fatalf("fresh object must not be reset, resets=%d", obj.resets)
	}
}

func TestPut_CallsResetSynchronously(t *testing.T) {
	p := pool.New(func() *testObj { return &testObj{} })

	obj := p.Get()
	obj.value = 42
	obj.data = append(obj.data, 1, 2, 3)

	p.Put(obj)

	if obj.value != 0 {
		t.Errorf("value not reset: %d", obj.value)
	}
	if len(obj.data) != 0 {
		t.Errorf("data not reset: len=%d", len(obj.data))
	}
	if obj.resets != 1 {
		t.Errorf("want exactly 1 Reset call, got %d", obj.resets)
	}
}
