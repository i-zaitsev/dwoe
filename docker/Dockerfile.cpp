# dwoe-agent:cpp adds a C++ toolchain (C++20 via Trixie GCC) to the shared agent base.
#
# Build:
# make image-cpp
#
FROM dwoe-base:latest

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    g++ \
    libc6-dev \
    make \
    cmake \
    pkg-config \
    gdb \
    valgrind \
    && rm -rf /var/lib/apt/lists/*

USER agent
WORKDIR /workspace
