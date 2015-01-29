# gengen - A generics code generator for Go #

People often lament the lack of generics in Go, and use it as an
[excuse to dismiss the
language](http://permalink.gmane.org/gmane.comp.lang.go.general/127789).
Yes, it is annoying that you often end up rewriting boilerplate.  And
yes, it is annoying that it's not possible to write a generic data
structure that can be type-checked at compile time.

However, we can use Go's [powerful source parsing and AST
representation packages](http://golang.org/pkg/go/) to build a program
that can translate generically-defined code into specifically typed
source code and compile that into our projects.

## How to use it ##

Get the `gengen` tool:

    $ go get github.com/alecthomas/gengen

Create a Go source file with a generic implementation.  For example,
this contrived linked-list implementation in `list.go`:

```go
package list

//go:generate gengen -o list_string.go -r "List -> StringList" list.go string
import "github.com/alecthomas/gengen/generic"

type List struct {
    data generic.T
    next *List
}

func (l *List) Prepend(d generic.T) *List {
    n := &List{
        data: d,
        next: l,
    }

    return n
}

func (l *List) Contains(d generic.T) bool {
    if l == nil {
        return false
    }

    for i := l; i != nil; i = i.next {
        if i.data == d {
            return true
        }
    }

    return false
}

func (l *List) Data() generic.T {
    if l == nil {
        // Return the zero value for generic.T, whatever type it ends
        // up becoming
        var zero generic.T
        return zero
    }

    return l.data
}

```

`generic.T` is simply `interface{}`.  This list implementation is
perfectly valid Go code and you could use it as-is, asserting types
at runtime.

However, you can generate a specifically typed version of this file by
running it through `gengen`:

    $ gengen list.go string > list_string.go

`list_string.go` will look like this:

```go
package list

type List struct {
    data string
    next *List
}

func (l *List) Prepend(d string) *List {
    n := &List{
        data: d,
        next: l,
    }

    return n
}

func (l *List) Contains(d string) bool {
    if l == nil {
        return false
    }

    for i := l; i != nil; i = i.next {
        if i.data == d {
            return true
        }
    }

    return false
}

func (l *List) Data() string {
    if l == nil {
        // Return the zero value for generic.T, whatever type it ends
        // up becoming (in this example, string)
        var zero string
        return zero
    }

    return l.data
}

```

The `generic` package also defines `generic.U` and `generic.V` as
additional generic types for cases when you want to support more than
one type.  Simply pass the additional types on the `gengen` command
line:

    $ gengen btree.go int string > btree_int_string.go

## Caveats ##

### Number of generic types ###

Currently `gengen` can support up to three generic types: `generic.T`,
`generic.U`, and `generic.V`.

### Naming ###

`gengen` does not currently do anything with naming of packages or
types.  This means that you can't compile both a generic version and a
type specific version in the same package without modification.
Fortunately, `gofmt -r` is perfect for this:

```go
type MyGenericSlice []generic.T
```

    $ gengen slice_generic.go string | gofmt -r 'MyGenericSlice -> MyStringSlice' > slice_string.go

```go
type MyStringSlice []string
```

### Using zero values ###

You may need to write code in a slightly different way than you
normally would for `interface{}` in order to support a wide range of
types.  For instance, in our `Data()` method, note that we cannot
simply `return nil` in the `l == nil` case because `nil` is not a
valid value for primitive types like `int`, `string`, etc.  Instead we
instantiate a variable of our generic type but do not assign to it,
ensuring that we always return the zero value for that type.

### Equality ###

Checking for equality in a generic implementation can be tricky, and
blindly checking `if x == y` [often will not work as you'd
hope](http://golang.org/ref/spec#Comparison_operators).  For things
like slices, it will not even compile.  If you need to check for
equality, you might want to create an `Equaler` interface, like so:

```go
type Equaler interface {
    Equal(other Equaler) bool
}
```

Define types that implement this interface:


```go
type intWithEqual int

func (i intWithEqual) Equal(other Equaler) bool {
    if i2, ok := other.(intWithEqual); ok {
        return i == i2
    }
    return false
}
```

```go
type Person struct {
    Name string
    SSN string
}

func (p *Person) Equal(other Equaler) bool {
    if p2, ok := other.(*Person); ok {
        return p.SSN == p2.SSN
    }
    return false
}
```

In your generic implementation, use the interface rather than
comparing directly:

```go
type MySlice []generic.T

func (s MySlice) Contains(e generic.T) bool {
    for _, e2 := range s {
        if e2.Equal(e) {
            return true
        }
    }
    return false
}
```

(Note that because `generic.T` does not embed the `Equaler` interface,
this code won't compile without being run through `gengen` first.)

Finally, generate your implementations:

    $ gengen myslice.go intWithEqual > myslice_int.go
    $ gengen myslice.go *Person > myslice_person.go

### Import and type naming inflexibility ###

The `gengen` tool looks through the source code for specific strings
in order to replace them in the AST.  Specifically, it looks for the
import `github.com/alecthomas/gengen/generic` and the types `generic.T`,
`generic.U`, and `generic.V`.  If you need to change these, you will
also have to change the `gengen.go` source.

## Origins ##

I had been mulling the idea of a generics generator for a while,
originally planning to use the `text/template` package.  However,
during a [panel discussion at
GopherCon](http://gophercon.sourcegraph.com/post/83845316771/panel-discussion-with-go-team-members)
in which generics inevitably came up, Rob Pike suggested manipulating
the AST for Go.  I began implementing this approach during the
GopherCon Hack Day on 26 April 2014.
