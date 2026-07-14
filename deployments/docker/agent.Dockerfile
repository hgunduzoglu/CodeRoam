FROM golang:1.26-alpine AS build
WORKDIR /src
COPY services/agent/go.mod ./
COPY services/agent/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-agent ./cmd/coderoam-agent

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-agent /usr/local/bin/coderoam-agent
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/coderoam-agent"]
