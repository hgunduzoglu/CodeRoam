module github.com/hgunduzoglu/coderoam/protocol/compat

go 1.26.0

require (
	github.com/hgunduzoglu/coderoam/protocol/gen/go v0.0.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/hgunduzoglu/coderoam/protocol/gen/go => ../gen/go
