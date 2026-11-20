package config

import (
	"flag"
)

// OptionalString is a flag.Value
type OptionalString struct {
	Value *string
}

func (o *OptionalString) String() string {
	if o.Value == nil {
		return ""
	}
	return *o.Value
}

func (o *OptionalString) Set(s string) error {
	v := s
	o.Value = &v
	return nil
}

func (o *OptionalString) IsSet() bool { return o.Value != nil }

// OptionalStringFlag registers an optional string flag on fs and returns
// the wrapper. Use Value and IsSet after fs.Parse.
func OptionalStringFlag(fs *flag.FlagSet, name, usage string) *OptionalString {
	o := &OptionalString{}
	fs.Var(o, name, usage)
	return o
}

// OptionalSecret is like OptionalString but never exposes its value via
// String().
type OptionalSecret struct {
	Value *string
}

func (o *OptionalSecret) String() string { return "" }

func (o *OptionalSecret) Set(s string) error {
	v := s
	o.Value = &v
	return nil
}

func (o *OptionalSecret) IsSet() bool { return o.Value != nil }

// OptionalSecretFlag registers an optional secret flag on fs and returns
// the wrapper.
func OptionalSecretFlag(fs *flag.FlagSet, name, usage string) *OptionalSecret {
	o := &OptionalSecret{}
	fs.Var(o, name, usage)
	return o
}
