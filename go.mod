module github.com/IAM-timmy1t/Quant_WebWork_GO

go 1.21

// Handle import path issues with replace directives
replace (
	github.com/IAM-timmy1t/Quant_WebWork_GO => ./
	google.golang.org/grpc/web => github.com/improbable-eng/grpc-web v0.15.0
)

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/graphql-go/graphql v0.8.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1
	github.com/prometheus/client_golang v1.18.0
	github.com/stretchr/testify v1.8.4
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe
	google.golang.org/grpc v1.62.0
	google.golang.org/grpc/web v0.0.0-00010101000000-000000000000
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240125205218-1f4bbc51befe // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
