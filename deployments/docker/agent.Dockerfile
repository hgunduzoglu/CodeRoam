FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY packages/go/cryptox/go.mod ./packages/go/cryptox/
COPY services/agent/go.mod services/agent/go.sum ./services/agent/
WORKDIR /src/services/agent
RUN go mod download
WORKDIR /src
COPY packages/go/cryptox/ ./packages/go/cryptox/
COPY services/agent/ ./services/agent/
WORKDIR /src/services/agent
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-agent ./cmd/coderoam-agent

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-agent /usr/local/bin/coderoam-agent
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/coderoam-agent"]
