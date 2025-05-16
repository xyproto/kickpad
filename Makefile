.PHONY: clean

all:
	go build -tags=enable_cimgui_sdl2 -mod=mod -trimpath -v

windows:
	x86_64-w64-mingw32-windres kickpad.rc -O coff -o kickpad.syso
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ HOST=x86_64-w64-mingw32 go build -ldflags "-s -w -H=windowsgui -extldflags='-static -L/usr/include'" -p 4 -v -o kickpad.exe
	rm kickpad.syso

clean:
	rm -f kickpad
