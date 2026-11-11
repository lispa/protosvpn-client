FROM ubuntu:24.04
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends nsis p7zip-full && rm -rf /var/lib/apt/lists/*
WORKDIR /src/build/windows
ENTRYPOINT ["makensis"]
