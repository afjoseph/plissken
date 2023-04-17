module github.com/afjoseph/plissken-auth-server

go 1.18

require (
	github.com/afjoseph/plissken-protocol v0.0.0-00010101000000-000000000000
	github.com/alicebob/miniredis/v2 v2.30.1
	github.com/cloudflare/circl v1.3.2
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.7.7
	github.com/go-redis/redis/v8 v8.11.4
	github.com/gopherjs/gopherjs v0.0.0-20220417154020-410b52891213
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/bwesterb/go-ristretto v1.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/yuin/gopher-lua v1.1.0 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace github.com/afjoseph/plissken-protocol => ../protocol-lib
