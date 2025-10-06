ARG GOLANG_VERSION

###########################################################
FROM golang:$GOLANG_VERSION AS builder

WORKDIR /build
COPY *.go go.mod go.sum ./
RUN go build -o user-model-metrics-webhook .

###########################################################
# The run stage
FROM debian:stable-slim
WORKDIR /app
COPY --from=builder /build/user-model-metrics-webhook /app/user-model-metrics-webhook

ENTRYPOINT ["/app/user-model-metrics-webhook"]
CMD []