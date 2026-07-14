FROM golang:1.26-alpine AS build
WORKDIR /src
COPY services/relay/go.mod ./
COPY services/relay/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-relay ./cmd/relay

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-relay /usr/local/bin/coderoam-relay
USER nonroot:nonroot
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/coderoam-relay"]
