module github.com/PoinkCorporation/SSO-service

go 1.26.3

require (
	github.com/PoinkCorporation/Permissions-service v0.0.0-20260531121918-fc2638a301b5
	github.com/brianvoe/gofakeit/v6 v6.28.0
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/jackc/pgconn v1.14.3
	github.com/jackc/pgx/v4 v4.18.3
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.45.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)

replace github.com/PoinkCorporation/Permissions-service => ../permissions-service
