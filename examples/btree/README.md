This btree module originally comes from
[github.com/cznic/b](https://github.com/cznic/b).  It deals with
`interface{}` but includes a `make generic` rule that replaces types
with `KEY` and `VALUE` strings, which you are then supposed to replace
with the types you want.

I have taken that and replaced `KEY` with `generic.T` and `VALUE` with
`generic.U`.

Use `gengen` to generate a type-specific instance:

    $ gengen btree int string > main/btree.go
    $ go run main/*.go