# synth

Generate audio samples.

* This package is a bit experimental and a work in progress.
* Includes a `kick` utility for generating kick drum samples, plus `rms` and `linear` for mixing audio and `sweep` for making a detuned synth sample. They are all in the `cmd` directory.
* Used by the [Kickpad](https://github.com/xyproto/kickpad) application.
* `ffplay` is needed for playing samples with this package, unless it is built with the `sdl2` go build tag (`go build -tags sdl2`).

## Build flags

* Build with `go build -tags sdl2` to use SDL2 instead of using GLFW and the `ffplay` command.

## General info

* License: MIT
* Version: 1.6.0
