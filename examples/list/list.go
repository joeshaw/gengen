package main

import (
	"fmt"

	"github.com/joeshaw/gengen/generic"
)

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
		// This type of equality check is not generically safe,
		// but will work fine for all value types.  See the
		// caveats section in the README.
		if i.data == d {
			return true
		}
	}
	return false
}

func (l *List) Data() generic.T {
	if l == nil {
		var x generic.T
		return x
	}

	return l.data
}

func main() {
	var l *List
	fmt.Println(l.Contains(456), l.Data())

	l = l.Prepend(123)
	fmt.Println(l.Contains(456), l.Data())

	l = l.Prepend(456)
	fmt.Println(l.Contains(456), l.Data())

	l = l.Prepend(789)
	fmt.Println(l.Contains(789), l.Data())
}
