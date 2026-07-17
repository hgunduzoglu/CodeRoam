FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY packages/go/cryptox/go.mod ./packages/go/cryptox/
COPY packages/go/postgresx/go.mod packages/go/postgresx/go.sum ./packages/go/postgresx/
COPY services/control-plane/go.mod services/control-plane/go.sum ./services/control-plane/
WORKDIR /src/services/control-plane
RUN go mod download
WORKDIR /src
COPY packages/go/cryptox/ ./packages/go/cryptox/
COPY packages/go/postgresx/ ./packages/go/postgresx/
COPY services/control-plane/ ./services/control-plane/
WORKDIR /src/services/control-plane
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-api /usr/local/bin/coderoam-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/coderoam-api"]
