#!/usr/bin/env bash

set -e

PLATFORMS=(
  darwin/amd64
  freebsd/386
  freebsd/amd64
  freebsd/arm
  freebsd/arm64
  linux/386
  linux/amd64
  linux/arm
  linux/arm64
  linux/ppc64
  linux/ppc64le
  linux/mips
  linux/mipsle
  linux/mips64
  linux/mips64le
)

CGO_ENABLE_PLATFORMS=(
    linux/amd64
)

getos() {
  local IFS=/ && read -ra array <<< "$1" && echo "${array[0]}"
}

getarch() {
  local IFS=/ && read -ra array <<< "$1" && echo "${array[1]}"
}

WHICH="$1"

VERSION="${TRAVIS_TAG:-$(git describe --tags --always --dirty)}"

GOLDFLAGS=${GLDFLAGS:-}
GOLDFLAGS="$GOLDFLAGS -w -s -X main.version=$VERSION"

build() {

  if printf '%s\n' "${CGO_ENABLE_PLATFORMS[@]}" | grep -q "^${GOOS}/${GOARCH}$"; then
      CGO_ENABLED=1
  else
      CGO_ENABLED=0
  fi

  echo "Building ${GOOS}/${GOARCH}"
  echo "SET CGO_ENABLE $CGO_ENABLED"
  GOOS="$1" GOARCH="$2" go build -ldflags "${GOLDFLAGS}" -o "$3" "github.com/netdata/go.d.plugin/cmd/godplugin"
}

build_all_platforms() {
  for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=$(getos "$PLATFORM")
    GOARCH=$(getarch "$PLATFORM")
    FILE="bin/go.d.plugin-${VERSION}.${GOOS}-${GOARCH}"

    build "$GOOS" "$GOARCH" "$FILE"

    ARCHIVE="${FILE}.tar.gz"
    tar -C bin -cvzf "${ARCHIVE}" "${FILE/bin\//}"
    rm "${FILE}"
  done
}

build_specific_platform() {
  GOOS=$(getos "$1")
  GOARCH=$(getarch "$1")
  : "${GOARCH:=amd64}"

  build "$GOOS" "$GOARCH" bin/godplugin
}

build_current_platform() {
  eval "$(go env | grep -e "GOHOSTOS" -e "GOHOSTARCH")"
  GOOS=${GOOS:-$GOHOSTOS}
  GOARCH=${GOARCH:-$GOHOSTARCH}

  build "$GOOS" "$GOARCH" bin/godplugin
}

echo "Building binaries for version: $VERSION"
if [[ "$WHICH" == "all" ]]; then
  build_all_platforms
elif [[ -n "$WHICH" ]]; then
  build_specific_platform "$WHICH"
else
  build_current_platform
fi
