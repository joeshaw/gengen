package main

import (
	"fmt"
)

func main() {
	t := TreeNew(func(a, b int) int {
		return a - b
	})

	t.Set(5, "five")
	t.Set(10, "ten")
	t.Set(1, "one")

	k, v := t.First()
	fmt.Println(k, v)
}
