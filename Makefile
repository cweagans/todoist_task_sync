
# Don't do anything unless told specifically what to do.
.PHONY: target-required
target-required:
	@echo "You must specify a make target"
	exit 1

deps:
	go mod vendor

linux:
	GOOS=linux GOARCH=amd64 go build -o ./bin/linux-amd64/ttsync

darwin:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/darwin-amd64/ttsync

windows:
	GOOS=windows GOARCH=amd64 go build -o ./bin/windows-amd64/ttsync

bin: deps linux darwin windows
