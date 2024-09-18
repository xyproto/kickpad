# synth

Generate audio samples.

* This package is a bit experimental and a work in progress.
* Includes a `kick` utility for generating kick drum samples, plus `rms` and `linear` for mixing audio and `sweep` for making a detuned synth sample. They are all in the `cmd` directory.
* Used by the [Kickpad](https://github.com/xyproto/kickpad) application.
* `ffplay` is needed for playing samples with this package, unless it is built with the `sdl2` go build tag (`go build -tags sdl2`).

## Free sample pack

`synth` was used to create this CC0 licensed kick drum sample pack which can be downloaded here (note that the `.wav` filenames are not as meaningful as I originally indended them to be).

* https://github.com/xyproto/synth/releases/download/v1.4.4/afr-kicks-2024.zip

## Build flags

* Build with `go build -tags sdl2` to use SDL2 instead of using GLFW and the `ffplay` command.

## General info

* License: MIT
* Version: 1.5.3
