set shell := ["bash", "-euo", "pipefail", "-c"]

version := "v0.34.1"
ldflags  := "-s -w -X github.com/taito-project/taito/cmd.Version=" + version

default:
    @just --list

build:
    go build -o taito .

fmt:
    go fmt ./...

test:
    go test ./...

complexity threshold="15":
    go run github.com/uudashr/gocognit/cmd/gocognit@latest -over {{threshold}} .

clean:
    rm -rf ./packaging/target/*
    rm -rf ./packaging/taito/binaries/*

package: clean
    # Build binaries into packaging/taito/binaries/
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags='{{ldflags}}' -o ./packaging/taito/binaries/taito-darwin-arm64 .
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='{{ldflags}}' -o ./packaging/taito/binaries/taito-darwin-amd64 .
    CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -ldflags='{{ldflags}}' -o ./packaging/taito/binaries/taito-linux-amd64 .
    CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -ldflags='{{ldflags}}' -o ./packaging/taito/binaries/taito-linux-arm64 .
    chmod +x ./packaging/taito/binaries/* ./packaging/taito/bin/taito.js

    # Copy standalone binaries to target/ for release artifacts
    mkdir -p ./packaging/target
    cp ./packaging/taito/binaries/* ./packaging/target/

    # Build Linux packages for amd64 and arm64
    VERSION={{version}} NFPM_ARCH=amd64 envsubst < ./packaging/nfpm.yaml > ./packaging/nfpm-rendered.yaml
    nfpm pkg --packager rpm -f ./packaging/nfpm-rendered.yaml --target ./packaging/target/
    nfpm pkg --packager apk -f ./packaging/nfpm-rendered.yaml --target ./packaging/target/
    nfpm pkg --packager deb -f ./packaging/nfpm-rendered.yaml --target ./packaging/target/
    VERSION={{version}} NFPM_ARCH=arm64 envsubst < ./packaging/nfpm.yaml > ./packaging/nfpm-rendered.yaml
    nfpm pkg --packager rpm -f ./packaging/nfpm-rendered.yaml --target ./packaging/target/
    nfpm pkg --packager apk -f ./packaging/nfpm-rendered.yaml --target ./packaging/target/
    nfpm pkg --packager deb -f ./packaging/nfpm-rendered.yaml --target ./packaging/target/
    rm -f ./packaging/nfpm-rendered.yaml

    # Generate checksums
    cd packaging/target && sha256sum * > checksum.txt
