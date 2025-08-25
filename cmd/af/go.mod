module github.com/agentflow/agentflow/cmd/af

go 1.23.0

replace github.com/agentflow/agentflow => ../..

require (
	github.com/agentflow/agentflow v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.5.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
