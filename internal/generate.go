package gateway

//go:generate sh -c "mkdir -p ../gen/proto && protoc --proto_path=../proto --go_out=../gen/proto --go_opt=paths=source_relative --go-grpc_out=../gen/proto --go-grpc_opt=paths=source_relative common.proto course.proto score.proto"
