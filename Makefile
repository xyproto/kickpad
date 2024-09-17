.PHONY: clean

# This Makefile is a work in progress

all: kickpad

kickpad: main.go
	go build -trimpath -v -ldflags "-linkmode 'external' -extldflags '-static'"

kickpad.exe: main.go
	go build -ldflags "-s -w -H=windowsgui -extldflags=-static" .

kickpad_macos: main.go
	 go build -ldflags "-s -w" . -o kickpad_macos

clean:
	rm -f kickpad kickpad.exe kickpad_macos
