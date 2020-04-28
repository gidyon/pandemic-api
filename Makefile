PROJECT_NAME=pandemic-api
SERVICE_OUT := server.bin
PKG := github.com/gidyon/pandemic-api
SERVICE_PKG_BUILD := ${PKG}/cmd/gateway
API_IN_PATH := api/proto
API_OUT_PATH := pkg/api
SWAGGER_DOC_OUT_PATH := api/swagger

setup_dev: ## start development databases
	cd deployments/docker-compose && docker-compose up -d

teardown_dev: ## stop development databases
	cd deployments/docker-compose && docker-compose down

run: ## run compiled binary
	./$(SERVICE_OUT)

run_gateway: ## go run server
	go run cmd/gateway/*.go

run_messaging:
	cd cmd/messaging && go build && FCM_SERVER_KEY=abc ./messaging -config-file=/home/gideon/go/src/github.com/gidyon/pandemic-api/configs/messaging_dev.yml

run_tracing:
	cd cmd/tracing && go build && ./tracing -config-file=/home/gideon/go/src/github.com/gidyon/pandemic-api/configs/tracing_dev.yml

run_location:
	cd cmd/location && go build && ./location -config-file=/home/gideon/go/src/github.com/gidyon/pandemic-api/configs/location_dev.yml
# ============================================================================== 
proto_compile_location:
	protoc -I=$(API_IN_PATH) -I=third_party --go_out=plugins=grpc:$(API_OUT_PATH)/location locationv2.proto &&\
	protoc -I=$(API_IN_PATH) -I=third_party --grpc-gateway_out=logtostderr=true:$(API_OUT_PATH)/location locationv2.proto &&\
	protoc -I=$(API_IN_PATH) -I=third_party --swagger_out=logtostderr=true:$(SWAGGER_DOC_OUT_PATH) locationv2.proto

compile_gateway:
	go build -i -v -o gateway $(SERVICE_PKG_BUILD)

docker_build: ## Create a docker image for the service
ifdef tag
	@docker build -t gidyon/$(PROJECT_NAME):$(tag) .
else
	@docker build -t gidyon/$(PROJECT_NAME):latest .
endif

docker_tag: ## Tag image
ifdef tag
	@docker tag gidyon/$(PROJECT_NAME):$(tag) gidyon/$(PROJECT_NAME):$(tag)
else
	@docker tag gidyon/$(PROJECT_NAME):latest gidyon/$(PROJECT_NAME):latest
endif

docker_push: ## Push image
ifdef tag
	@docker push gidyon/$(PROJECT_NAME):$(tag)
else
	@docker push gidyon/$(PROJECT_NAME):latest
endif

build_and_push: compile_gateway docker_build docker_tag docker_push

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
