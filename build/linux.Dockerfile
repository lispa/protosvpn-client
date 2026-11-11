FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates pkg-config gcc libc6-dev libgtk-3-dev libwebkit2gtk-4.1-dev xvfb xauth dbus-x11 && rm -rf /var/lib/apt/lists/*
ENV PATH="/opt/go/bin:${PATH}" GOTOOLCHAIN=local GOTELEMETRY=off GOMAXPROCS=2 CGO_ENABLED=1 GOCACHE=/tmp/go-build
WORKDIR /src
