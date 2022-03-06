module github.com/arindas/proglog/internal/log

go 1.17

require (
	github.com/arindas/proglog/api/log_v1 v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	github.com/tysonmote/gommap v0.0.1
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/arindas/proglog/api/log_v1 => ../../api/v1
