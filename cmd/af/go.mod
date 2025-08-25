module github.com/agentflow/agentflow/cmd/af

go 1.23.0

replace github.com/agentflow/agentflow => ../..

require (
	github.com/agentflow/agentflow v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.5.1
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/text v0.26.0 // indirect
)
