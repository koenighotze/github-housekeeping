FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /bin/housekeeping ./cmd/housekeeping

FROM alpine:3.20
RUN apk add --no-cache curl unzip ca-certificates

ARG OP_VERSION=v2.28.0
RUN curl -sSfL "https://cache.agilebits.com/dist/1P/op2/pkg/${OP_VERSION}/op_linux_amd64_${OP_VERSION}.zip" \
      -o /tmp/op.zip && \
    unzip /tmp/op.zip op -d /usr/local/bin && \
    chmod +x /usr/local/bin/op && \
    rm /tmp/op.zip && \
    apk del curl unzip

COPY --from=build /bin/housekeeping /usr/local/bin/housekeeping

ENTRYPOINT ["housekeeping"]
CMD ["--config", "/config/config.yaml"]
