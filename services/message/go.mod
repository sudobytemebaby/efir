module github.com/sudobytemebaby/efir/services/message

go 1.25.0

require (
	buf.build/go/protovalidate v1.1.3
	github.com/google/uuid v1.6.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/nats-io/nats.go v1.49.0
	github.com/stretchr/testify v1.11.1
	github.com/sudobytemebaby/efir/services/shared v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.79.2
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260209202127-80ab13bee0bf.1 // indirect
	cel.dev/expr v0.25.1 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/cel-go v0.27.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/exp v0.0.0-20250813145105-42675adae3e6 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)

replace github.com/sudobytemebaby/efir/services/shared => ../shared
