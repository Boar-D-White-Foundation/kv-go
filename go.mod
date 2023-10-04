module github.com/Boar-D-White-Foundation/kv-go

go 1.21.1

replace (
	github.com/Boar-D-White-Foundation/kv-go/cli => ./cli
	github.com/Boar-D-White-Foundation/kv-go/monitoring => ./monitoring
	github.com/Boar-D-White-Foundation/kv-go/node => ./node
	github.com/Boar-D-White-Foundation/kv-go/server => ./server
	github.com/Boar-D-White-Foundation/kv-go/tests => ./tests
)

require (
	github.com/Boar-D-White-Foundation/kv-go/cli v0.0.1
	github.com/Boar-D-White-Foundation/kv-go/monitoring v0.0.1
	github.com/Boar-D-White-Foundation/kv-go/node v0.0.1
	github.com/Boar-D-White-Foundation/kv-go/server v0.0.1
	github.com/Boar-D-White-Foundation/kv-go/tests v0.0.1
)