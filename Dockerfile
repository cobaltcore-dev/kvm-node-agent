# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.3-alpine3.22 AS builder

RUN apk add --no-cache --no-progress ca-certificates gcc git make musl-dev

COPY . /src
ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION # provided to 'make install'
RUN make -C /src install PREFIX=/pkg GOTOOLCHAIN=local

################################################################################

FROM alpine:3.22

# upgrade all installed packages to fix potential CVEs in advance
# also remove apk package manager to hopefully remove dependency on OpenSSL 🤞
RUN apk upgrade --no-cache --no-progress \
  && apk del --no-cache --no-progress apk-tools alpine-keys alpine-release libc-utils

COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=builder /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=builder /pkg/ /usr/
# make sure all binaries can be executed
RUN set -x \
  && manager --version 2>/dev/null

ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION
LABEL source_repository="https://github.com/cobaltcore-dev/kvm-node-agent" \
  org.opencontainers.image.url="https://github.com/cobaltcore-dev/kvm-node-agent" \
  org.opencontainers.image.created=${BININFO_BUILD_DATE} \
  org.opencontainers.image.revision=${BININFO_COMMIT_HASH} \
  org.opencontainers.image.version=${BININFO_VERSION}

# Hardcoded to kvm-node-agent user/group on host
USER 42438:42438
WORKDIR /
ENTRYPOINT [ "/usr/bin/manager" ]
