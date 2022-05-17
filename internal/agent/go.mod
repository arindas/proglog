module github.com/arindas/proglog/internal/agent

go 1.18

replace github.com/arindas/proglog/internal/auth => ../auth

replace github.com/arindas/proglog/internal/discovery => ../discovery

replace github.com/arindas/proglog/internal/log => ../log

replace github.com/arindas/proglog/internal/server => ../server

replace github.com/arindas/proglog/internal/config => ../config

replace github.com/arindas/proglog/api/log_v1 => ../../api/v1

require (
	github.com/arindas/proglog/api/log_v1 v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/auth v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/config v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/discovery v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/log v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/server v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	github.com/travisjeffery/go-dynaport v1.0.0
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.45.0
)

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/armon/go-metrics v0.3.8 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/casbin/casbin v1.9.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-hclog v0.9.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-sockaddr v1.0.0 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/hashicorp/memberlist v0.3.0 // indirect
	github.com/hashicorp/raft v1.3.9 // indirect
	github.com/hashicorp/raft-boltdb/v2 v2.2.2 // indirect
	github.com/hashicorp/serf v0.9.7 // indirect
	github.com/miekg/dns v1.1.41 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/tysonmote/gommap v0.0.1 // indirect
	go.etcd.io/bbolt v1.3.5 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210410081132-afb366fc7cd1 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
