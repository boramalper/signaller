.PHONY: signaller vet staticcheck format check-formatting

signaller:
	go install github.com/boramalper/signaller/signallerd

vet:
	go vet github.com/boramalper/signaller/signallerd/...

staticcheck:
	staticcheck github.com/boramalper/signaller/signallerd/...

format:
	gofmt -w ${GOPATH}/src/github.com/boramalper/signaller/signallerd/

# Formatting Errors
#     Since gofmt returns zero even if there are files to be formatted, we use:
#
#       ! gofmt -d ${GOPATH}/path/ 2>&1 | read
#
#     to return 1 if there are files to be formatted, and 0 if not.
#     https://groups.google.com/forum/#!topic/Golang-Nuts/pdrN4zleUio
#
# How can I use Bash syntax in Makefile targets?
#     Because `read` is a bash command.
#     https://stackoverflow.com/a/589300/4466589
#
check-formatting: SHELL:=/bin/bash   # HERE: this is setting the shell for check-formatting only
check-formatting:
	! gofmt -l ${GOPATH}/src/github.com/boramalper/signaller/signallerd/ 2>&1 | tee /dev/fd/2 | read
