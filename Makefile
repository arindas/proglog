CONFIG_PATH=${HOME}/.config/proglog

.PHONY: clean-conf
clean-conf:
	rm -rf ${CONFIG_PATH}

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
gencert: init
	cfssl gencert \
		-initca ./testconf/certconf/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=./testconf/certconf/ca-config.json \
		-profile=server \
		./testconf/certconf/server-csr.json | cfssljson -bare server

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=./testconf/certconf/ca-config.json \
		-profile=client \
		-cn="root" \
		./testconf/certconf/client-csr.json | cfssljson -bare root-client
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=./testconf/certconf/ca-config.json \
		-profile=client \
		-cn="nobody" \
		./testconf/certconf/client-csr.json | cfssljson -bare nobody-client

	mv *.pem *.csr ${CONFIG_PATH}

compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.

test:
	go test -race ./...
