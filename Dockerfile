ARG GOLANG_VERSION

###########################################################
# The build stage
FROM golang:$GOLANG_VERSION AS builder

WORKDIR /build
COPY *.go go.mod go.sum ./
RUN go build -o user-model-metrics-webhook .

###########################################################
# The run stage
FROM debian:stable-slim
WORKDIR /app
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update -qq \
    && apt-get install -qq --no-install-recommends wget \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
COPY --from=builder /build/user-model-metrics-webhook /app/user-model-metrics-webhook

ENTRYPOINT ["/app/user-model-metrics-webhook"]
CMD []