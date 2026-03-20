module github.com/sudobytemebaby/efir/services/websocket

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/nats-io/nats.go v1.49.0
	github.com/stretchr/testify v1.11.1
	github.com/sudobytemebaby/efir/services/shared v0.0.0
	github.com/valkey-io/valkey-go v1.0.73
	nhooyr.io/websocket v1.8.10
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)

replace github.com/sudobytemebaby/efir/services/shared => ../shared
