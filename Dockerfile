ARG CLAUDE_VERSION=2.1.201
ARG CLAUDE_CONFIG_DIR=/config
ARG TWEAKCC_VERSION=4.0.11

##
## goshim builder:
##  Cross-compiles the goshim binary natively on $BUILDPLATFORM, regardless
##  of the target arch — Go's cross-compiler emits the right ELF without
##  ever running foreign-arch code under QEMU.
##
FROM --platform=$BUILDPLATFORM golang:1.25-alpine3.21 AS goshim-builder
ARG TARGETOS
ARG TARGETARCH
# llvm provides llvm-strip, which (unlike binutils strip) handles any target arch
RUN apk add --no-cache llvm
WORKDIR /src
COPY goshim/ ./
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /goshim . && \
    llvm-strip --strip-all /goshim

##
## tweakcc builder:
##  Downloads tweakcc source, builds it, and compiles to a static binary via Bun.
##
FROM oven/bun:alpine AS tweakcc-builder
ARG TWEAKCC_VERSION

RUN apk add --no-cache unzip python3 make g++

ADD https://github.com/Piebald-AI/tweakcc/archive/refs/tags/v${TWEAKCC_VERSION}.zip /tmp/tweakcc.zip
RUN unzip /tmp/tweakcc.zip -d /tmp && mv /tmp/tweakcc-${TWEAKCC_VERSION} /usr/local/lib/tweakcc

WORKDIR /usr/local/lib/tweakcc
RUN bun install
RUN bun add react-devtools-core
RUN bun run build:dev
RUN bun build ./dist/index.mjs --compile --outfile /tweakcc

##
## claude builder:
##  Fetches and installs claude-code
##
FROM alpine:3.21 AS builder
ARG CLAUDE_VERSION
ARG CLAUDE_CONFIG_DIR

RUN apk add --no-cache curl bash

ENV HOME=/home/agent

RUN mkdir -p ${HOME} && chmod 0755 ${HOME} && \
    # .claude.json MUST exist within our home directory
    #  and BOTH the source and destination file MUST be called ".claude.json"
    ln -s ${CLAUDE_CONFIG_DIR}/.claude.json ${HOME}/.claude.json

ADD https://claude.ai/install.sh /tmp/install.sh
RUN bash /tmp/install.sh ${CLAUDE_VERSION}

##
## Our final image:
##  Contains only claude-code, goshim, and the bootstrap template
##
FROM scratch
ARG CLAUDE_CONFIG_DIR

ENV HOME=/home/agent \
    CLAUDE_CONFIG_DIR=${CLAUDE_CONFIG_DIR} \
    # During a bootstrap and our CLAUDE_CONFIG_DIR is empty
    # it will be populated with the content of TEMPLATE_DIR
    TEMPLATE_DIR=/template-config
# Supresses some warnings from claude
ENV PATH=${HOME}/.local/bin:/usr/local/bin \
    CLAUDE_BIN=${HOME}/.local/bin/claude

# A place for claude to store your session and your anthropic credentials
#  To bootstrap this MUST be a copy of template-config
VOLUME ${CLAUDE_CONFIG_DIR}

# An empty directory that claude can "fully trust"
#  Don't put anything in here.  Use MCPs to read/write your files
#  from a TRUSTED environment.
WORKDIR /ai_contained

COPY --from=builder /lib/ld-musl-*.so.1 /lib/
COPY --from=builder /lib/libc.musl-*.so.1 /lib/
COPY --from=goshim-builder /goshim /usr/local/bin/shim_claude
COPY --from=tweakcc-builder /tweakcc /usr/local/bin/tweakcc
COPY --from=tweakcc-builder /usr/lib/libstdc++.so* /usr/lib/
COPY --from=tweakcc-builder /usr/lib/libgcc_s.so* /usr/lib/
COPY --from=tweakcc-builder /usr/local/lib/tweakcc/node_modules/wasmagic/dist/libmagic-wrapper.wasm /usr/local/lib/tweakcc/node_modules/wasmagic/dist/libmagic-wrapper.wasm
COPY /template-config ${TEMPLATE_DIR}
COPY --from=builder ${HOME} ${HOME}

# You should run claude with the same uid/gid as the /config volume
#  Which will probably be yours
USER 65533:65533

ENTRYPOINT ["/usr/local/bin/shim_claude"]
