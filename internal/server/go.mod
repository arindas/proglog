module github.com/arindas/proglog/internal/server

go 1.17

replace github.com/arindas/proglog/api/log_v1 => ../../api/v1

replace github.com/arindas/proglog/internal/log => ../log

replace github.com/arindas/proglog/internal/config => ../config

replace github.com/arindas/proglog/internal/auth => ../auth

require (
	github.com/arindas/proglog/api/log_v1 v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/auth v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/config v0.0.0-00010101000000-000000000000
	github.com/arindas/proglog/internal/log v0.0.0-00010101000000-000000000000
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.45.0
)

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/casbin/casbin v1.9.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tysonmote/gommap v0.0.1 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)
