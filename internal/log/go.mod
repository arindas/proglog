module github.com/arindas/proglog/internal/log

go 1.17

require (
	github.com/arindas/proglog/api/log_v1 v0.0.0-00010101000000-000000000000
	github.com/hashicorp/raft v1.3.9
	github.com/hashicorp/raft-boltdb v0.0.0-20220329195025-15018e9b97e0
	github.com/stretchr/testify v1.7.0
	github.com/tysonmote/gommap v0.0.1
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
)

require (
	github.com/armon/go-metrics v0.3.8 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/hashicorp/go-hclog v0.9.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/arindas/proglog/api/log_v1 => ../../api/v1
