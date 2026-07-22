FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY packages/go/ids/go.mod ./packages/go/ids/
COPY packages/go/postgresx/go.mod packages/go/postgresx/go.sum ./packages/go/postgresx/
COPY services/worker/go.mod services/worker/go.sum ./services/worker/
WORKDIR /src/services/worker
RUN go mod download
WORKDIR /src
COPY packages/go/ids/ ./packages/go/ids/
COPY packages/go/postgresx/ ./packages/go/postgresx/
COPY services/worker/ ./services/worker/
WORKDIR /src/services/worker
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-worker /usr/local/bin/coderoam-worker
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/coderoam-worker"]
