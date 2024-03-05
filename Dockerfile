ARG UPSTREAM_REPO
ARG UPSTREAM_TAG
ARG GO_VER
FROM ${UPSTREAM_REPO:-uselagoon}/commons:${UPSTREAM_TAG:-latest} as commons
FROM golang:${GO_VER:-1.21}-alpine3.18 as golang

RUN apk add --no-cache git
RUN go install github.com/a8m/envsubst/cmd/envsubst@v1.4.2

WORKDIR /app

COPY . ./

ARG BUILD
ARG GO_VER
ARG VERSION 
ENV BUILD=${BUILD} \
    GO_VER=${GO_VER} \
    VERSION=${VERSION}

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Do not force rebuild of up-to-date packages (do not use -a) and use the compiler cache folder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build \
    -ldflags="-s -w \
    -X github.com/uselagoon/build-deploy-tool/cmd.bdtBuild=${BUILD} \
    -X github.com/uselagoon/build-deploy-tool/cmd.goVersion=${GO_VER} \
    -X github.com/uselagoon/build-deploy-tool/cmd.bdtVersion=${VERSION} \
    -extldflags '-static'" \
    -o /app/build-deploy-tool .

# RUN go mod download
# RUN go build -o /app/build-deploy-tool

FROM docker:20.10.24

LABEL org.opencontainers.image.authors="The Lagoon Authors" maintainer="The Lagoon Authors"
LABEL org.opencontainers.image.source="https://github.com/uselagoon/lagoon-images" repository="https://github.com/uselagoon/lagoon-images"

ENV LAGOON=build-deploy-image

COPY --from=golang /go/bin/envsubst /bin/envsubst

ARG LAGOON_VERSION
ENV LAGOON_VERSION=$LAGOON_VERSION

# Copy commons files
COPY --from=commons /lagoon /lagoon
COPY --from=commons /bin/fix-permissions /bin/ep /bin/docker-sleep /bin/
COPY --from=commons /sbin/tini /sbin/
COPY --from=commons /home /home

RUN chmod g+w /etc/passwd \
    && mkdir -p /home

ENV TMPDIR=/tmp \
    TMP=/tmp \
    HOME=/home \
    # When Bash is invoked via `sh` it behaves like the old Bourne Shell and sources a file that is given in `ENV`
    ENV=/home/.bashrc \
    # When Bash is invoked as non-interactive (like `bash -c command`) it sources a file that is given in `BASH_ENV`
    BASH_ENV=/home/.bashrc

# Defining Versions
ENV KUBECTL_VERSION=v1.27.6 \
    HELM_VERSION=v3.13.0 \
    HELM_SHA256=138676351483e61d12dfade70da6c03d471bbdcac84eaadeb5e1d06fa114a24f

RUN apk add -U --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing aufs-util \
    && apk upgrade --no-cache openssh openssh-keygen openssh-client-common openssh-client-default \
    && apk add --no-cache openssl curl jq parallel bash git py-pip skopeo \
    && git config --global user.email "lagoon@lagoon.io" && git config --global user.name lagoon \
    && pip install shyaml yq \
    && curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
    && chmod +x /usr/bin/kubectl \
    && curl -Lo /usr/bin/yq3 https://github.com/mikefarah/yq/releases/download/3.3.2/yq_linux_amd64 \
    && chmod +x /usr/bin/yq3 \
    && curl -Lo /usr/bin/yq https://github.com/mikefarah/yq/releases/download/v4.35.2/yq_linux_amd64 \
    && chmod +x /usr/bin/yq \
    && curl -Lo /tmp/helm.tar.gz https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz \
    && echo "${HELM_SHA256}  /tmp/helm.tar.gz" | sha256sum -c -  \
    && mkdir /tmp/helm \
    && tar -xzf /tmp/helm.tar.gz -C /tmp/helm --strip-components=1 \
    && mv /tmp/helm/helm /usr/bin/helm \
    && chmod +x /usr/bin/helm \
    && rm -rf /tmp/helm*

RUN mkdir -p /kubectl-build-deploy/git
RUN mkdir -p /kubectl-build-deploy/lagoon

WORKDIR /kubectl-build-deploy/git

COPY legacy/docker-entrypoint.sh /lagoon/entrypoints/100-docker-entrypoint.sh
COPY legacy/build-deploy.sh /kubectl-build-deploy/build-deploy.sh
COPY legacy/build-deploy-docker-compose.sh /kubectl-build-deploy/build-deploy-docker-compose.sh

COPY legacy/scripts /kubectl-build-deploy/scripts

COPY legacy/helmcharts  /kubectl-build-deploy/helmcharts

ENV DBAAS_OPERATOR_HTTP=dbaas.lagoon.svc:5000

RUN curl -sSL https://github.com/uselagoon/lagoon-linter/releases/download/v0.8.0/lagoon-linter_0.8.0_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin lagoon-linter

# RUN curl -sSL https://github.com/uselagoon/build-deploy-tool/releases/download/v0.11.0/build-deploy-tool_0.11.0_linux_amd64.tar.gz \
#     | tar -xz -C /usr/local/bin build-deploy-tool
COPY --from=golang /app/build-deploy-tool /usr/local/bin/build-deploy-tool

# enable running unprivileged
RUN fix-permissions /home && fix-permissions /kubectl-build-deploy

ENTRYPOINT ["/sbin/tini", "--", "/lagoon/entrypoints.sh"]
CMD ["/kubectl-build-deploy/build-deploy.sh"]
