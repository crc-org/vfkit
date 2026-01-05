FROM registry.access.redhat.com/ubi10/go-toolset:latest AS build
WORKDIR /src/vfkit/vfkit

COPY . /src/vfkit/

RUN git config --global --add safe.directory /src/vfkit/vfkit

RUN CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o out/vfkit-amd64 ./cmd/vfkit
RUN CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o out/vfkit-arm64 ./cmd/vfkit

FROM scratch
COPY --from=build /src/vfkit/vfkit/out/vfkit-amd64 /releases/vfkit-amd64
COPY --from=build /src/vfkit/vfkit/out/vfkit-arm64 /releases/vfkit-arm64
