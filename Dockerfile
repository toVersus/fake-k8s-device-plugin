FROM cgr.dev/chainguard/go:1.20.3 as builder

WORKDIR /go/src/app
COPY . .

RUN CGO_ENABLED=0 go build .

FROM cgr.dev/chainguard/wolfi-base:latest
COPY --from=builder /go/src/app/fake-k8s-device-plugin /usr/bin/fake-k8s-device-plugin

ENTRYPOINT ["/usr/bin/fake-k8s-device-plugin"]
