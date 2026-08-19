FROM golang:1.26.6-alpine AS build
WORKDIR /src
COPY packages/go/ids/go.mod ./packages/go/ids/
COPY protocol/gen/go/go.mod protocol/gen/go/go.sum ./protocol/gen/go/
COPY services/relay/go.mod services/relay/go.sum ./services/relay/
WORKDIR /src/services/relay
RUN go mod download
WORKDIR /src
COPY packages/go/ids/ ./packages/go/ids/
COPY protocol/gen/go/ ./protocol/gen/go/
COPY services/relay/ ./services/relay/
WORKDIR /src/services/relay
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-relay ./cmd/relay

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-relay /usr/local/bin/coderoam-relay
USER nonroot:nonroot
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/coderoam-relay"]
