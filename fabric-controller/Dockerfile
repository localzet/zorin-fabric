FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY internal ./internal
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/fabric-controller ./cmd/fabric-controller \
 && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/fabric-edge-agent ./cmd/fabric-edge-agent \
 && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/fabric-bootstrap ./cmd/fabric-bootstrap \
 && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/fabric-trust-exchange ./cmd/fabric-trust-exchange \
 && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/fabric-health ./cmd/fabric-health

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tini \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --system --create-home --home /var/lib/zorin-fabric --shell /usr/sbin/nologin fabric
COPY --from=build /out/* /usr/local/bin/
USER fabric
WORKDIR /var/lib/zorin-fabric
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/fabric-controller"]
CMD ["--config", "/etc/zorin-fabric/fabric-controller.json"]
