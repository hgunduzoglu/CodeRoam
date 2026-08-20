module github.com/hgunduzoglu/coderoam/services/relay

go 1.26.0

require (
	github.com/hgunduzoglu/coderoam/packages/go/ids v0.0.0
	github.com/hgunduzoglu/coderoam/protocol/gen/go v0.0.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/hgunduzoglu/coderoam/packages/go/ids => ../../packages/go/ids

replace github.com/hgunduzoglu/coderoam/protocol/gen/go => ../../protocol/gen/go
