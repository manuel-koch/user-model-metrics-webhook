ARG GOLANG_VERSION

###########################################################
FROM golang:$GOLANG_VERSION AS builder

WORKDIR /build
COPY *.go go.mod go.sum ./
RUN go build -o user-metric-webhook .

###########################################################
# The run stage
FROM debian:stable-slim
WORKDIR /app
COPY --from=builder /build/user-metric-webhook /app/user-metric-webhook

ENTRYPOINT ["/app/user-metric-webhook"]
CMD []