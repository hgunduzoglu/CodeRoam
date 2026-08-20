module github.com/hgunduzoglu/coderoam/services/agent

go 1.26.0

require (
	github.com/hgunduzoglu/coderoam/packages/go/cryptox v0.0.0
	golang.org/x/sys v0.47.0
)

replace github.com/hgunduzoglu/coderoam/packages/go/cryptox => ../../packages/go/cryptox
