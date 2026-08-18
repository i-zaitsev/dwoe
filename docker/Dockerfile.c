# dwoe-agent:c adds a C toolchain (C23 via Trixie GCC) to the shared agent base.
#
# Build:
# make image-c
#
FROM dwoe-base:latest

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    make \
    cmake \
    pkg-config \
    gdb \
    valgrind \
    && rm -rf /var/lib/apt/lists/*

USER agent
WORKDIR /workspace
