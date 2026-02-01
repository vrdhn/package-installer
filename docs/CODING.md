# Coding style guidelines

Go:
 * Use all the feature of latest version of go.
 * code so that static type checking is done, i.e. avoid any as far as possible.
 * if 'any' can't be avoided, a runtime type checking with proper error is must.



Structural:
 * keep files small, around 250 lines is optimal
 * however the complete interface implementation of a struct should be single file
 * keep functions small, 15 lines , and up to 3 indentation level is optimal
 * avoid globals and init functions as far as possible.
 * prefer lots of packages, multi level directory
 * ideally, a package should provide implementation of few related interfaces.


Immutability:
 * have readonly interface, and extend it to a writable interface
 * readonly interface has ability to freeze, and to checkout the writable interface
 * checking out wr interface second time or from frozen struct is panic.

Exhaustive case:
 * ALWAYS add default:panic, if there is no default.


At package boundary:
 * try to use interfaces, rather than struct
 * try to define specific types instead of using primitive types
 * add impementation check _ : interface = type{} trick.


Precommit:
 * run `go fmt ./...`
 * run `go vet ./...` and fix issues
 * run `go test ./...` and fix issues
