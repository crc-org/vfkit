.PHONY: all
all: build codesign

.PHONY: codesign
codesign:
	codesign --entitlements vf.entitlements -s - ./vfkit

.PHONY: build
build:
	go build -o vfkit ./cmd/vfkit
