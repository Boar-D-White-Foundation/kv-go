package kvgo

import (
	"testing"

	"github.com/matryer/is"
)

func TestFoo(t *testing.T) {
	t.Parallel()

	is := is.New(t)
	is.Equal(Foo(), "bar")
}
