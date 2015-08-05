To run the generic version:

    $ go run slice.go -bool=true
    $ go run slice.go -int=5
    $ go run slice.go -string=foo

To generate the `string` version:

    $ gengen -o slice_string github.com/joeshaw/gengen/examples/slice string

To run the `string` version:

    $ go run slice_string/slice.go -string=foo

If you attempt to pass `-int` or `-bool` to the `string` version, it
will panic because the `Contains()` function is strictly defined to
take a `string` argument.
