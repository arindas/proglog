module github.com/arindas/proglog/internal/loadbalance

go 1.18

replace github.com/arindas/proglog/api/log_v1 => ../../api/v1

replace github.com/arindas/proglog/internal/log => ../log

replace github.com/arindas/proglog/internal/config => ../config

replace github.com/arindas/proglog/internal/auth => ../auth

replace github.com/arindas/proglog/internal/server => ../server

require (
	github.com/arindas/proglog/api/log_v1 v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/config v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/server v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.46.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
