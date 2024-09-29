.PHONY: clean

all:
	go build -tags=enable_cimgui_sdl2 -mod=mod -trimpath

clean:
	rm -f kickpad
