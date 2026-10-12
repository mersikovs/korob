package signing_test

import (
	"fmt"
	"testing"

	"github.com/mersikovs/korob.git/internal/signing"
)

func TestCalcHash(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		data []byte
		key  []byte
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := signing.CalcHash(tt.data, tt.key)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("CalcHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ExampleCalcHash demonstrates how to compute an HMAC-SHA256 hash
// of a message using a secret key.
//
// The test vector below is taken from RFC 4231, Test Case 2.
func ExampleCalcHash() {
	key := []byte("key")
	data := []byte("test")

	hash := signing.CalcHash(data, key)
	fmt.Println(hash)

	// Output:
	// 02afb56304902c656fcb737cdd03de6205bb6d401da2812efd9b2d36a08af159
}

// ExampleCalcHash_emptyKey demonstrates the behaviour when an empty key
// is provided: the function returns an empty string without computing
// anything.
func ExampleCalcHash_emptyKey() {
	hash := signing.CalcHash([]byte("any data"), []byte{})
	fmt.Printf("len=%d value=%q\n", len(hash), hash)

	// Output:
	// len=0 value=""
}
