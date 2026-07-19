module github.com/hgunduzoglu/coderoam/services/worker

go 1.26.0

require (
	github.com/hgunduzoglu/coderoam/packages/go/ids v0.0.0
	github.com/jackc/pgx/v5 v5.10.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace github.com/hgunduzoglu/coderoam/packages/go/ids => ../../packages/go/ids

replace github.com/hgunduzoglu/coderoam/packages/go/postgresx => ../../packages/go/postgresx
