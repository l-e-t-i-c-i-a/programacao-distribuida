#!/bin/bash
GITHUB_USERNAME=l-e-t-i-c-i-a
GITHUB_EMAIL=proj3705@gmail.com

# 1. MUDANÇA AQUI: O nome do serviço deve ser igual ao nome da pasta
SERVICE_NAME=order
RELEASE_VERSION=v1.2.3

# 2. MUDANÇA AQUI: Instalando os DOIS plugins necessários (protobuf e grpc)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

export PATH="$PATH:$(go env GOPATH)/bin"
# source ~/.zshrc

echo "Generating Go source code"
mkdir -p golang

# 3. O comando protoc
protoc --go_out=./golang \
  --go_opt=paths=source_relative \
  --go-grpc_out=./golang \
  --go-grpc_opt=paths=source_relative \
 ./${SERVICE_NAME}/*.proto

echo "Generated Go source code files"
ls -al ./golang/${SERVICE_NAME}

cd golang/${SERVICE_NAME}
go mod init \
  github.com/${GITHUB_USERNAME}/microservices-proto/golang/${SERVICE_NAME} || true
go mod tidy || true