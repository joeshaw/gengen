package main

import (
	"flag"
	"fmt"

	"github.com/alecthomas/gengen/generic"
)

type MySlice []generic.T

func (s MySlice) Contains(g generic.T) bool {
	for _, g2 := range s {
		if g == g2 {
			return true
		}
	}
	return false
}

func main() {
	boolFlag := flag.Bool("bool", false, "boolean value")
	intFlag := flag.Int("int", 0, "integer value")
	stringFlag := flag.String("string", "", "string value")

	flag.Parse()

	var iface interface{}
	if *boolFlag {
		iface = *boolFlag
	} else if *intFlag != 0 {
		iface = *intFlag
	} else if *stringFlag != "" {
		iface = *stringFlag
	} else {
		fmt.Println("Provide one of -bool, -int, or -string")
		return
	}

	var s MySlice
	var zero generic.T
	s = append(s, zero, zero, iface.(generic.T), zero, zero)
	fmt.Println(s)
	fmt.Println(s.Contains(iface.(generic.T)))
}
