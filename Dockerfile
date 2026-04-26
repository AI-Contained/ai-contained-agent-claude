ARG CLAUDE_VERSION=2.1.114
ARG CLAUDE_CONFIG_DIR=/config

##
## Our builder image:
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
##  Contains only claude-code
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

COPY /template-config ${TEMPLATE_DIR}
COPY --from=builder /lib/ld-musl-*.so.1 /lib/
COPY --from=builder /lib/libc.musl-*.so.1 /lib/
COPY --from=builder ${HOME} ${HOME}

# You should run claude with the same uid/gid as the /config volume
#  Which will probably be yours
USER 65533:65533

ENTRYPOINT ["/home/agent/.local/bin/claude"]
