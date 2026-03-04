#!/bin/bash

PLATFORMS=("windows/amd64" "windows/arm64" "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")

for platform in "${PLATFORMS[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name="rymcli-$GOOS-$GOARCH"
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    echo "Building for $GOOS/$GOARCH..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -o bin/$output_name
done