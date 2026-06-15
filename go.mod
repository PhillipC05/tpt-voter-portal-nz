module github.com/tpt-nz/tpt-voter-portal-nz

go 1.25.0

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/tpt-nz/nz-common v0.0.0-00010101000000-000000000000
	github.com/tpt-nz/realme-go v0.0.0-00010101000000-000000000000
)

require (
	github.com/beevik/etree v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/crewjam/httperr v0.2.0 // indirect
	github.com/crewjam/saml v0.4.14 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.3 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/redis/go-redis/v9 v9.20.1 // indirect
	github.com/russellhaering/goxmldsig v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace github.com/tpt-nz/nz-common => ./packages/nz-common

replace github.com/tpt-nz/realme-go => ./packages/realme-go
