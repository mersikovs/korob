package main

type fakeOS struct{}

func (fakeOS) Exit(code int) {}

func main() {
	os := fakeOS{}

	os.Exit(1) // no error
}
