PROJECT_NAME=pandemic-api
SERVICE_OUT := server.bin
PKG := github.com/gidyon/pandemic-api
SERVICE_PKG_BUILD := ${PKG}/cmd/server
API_IN_PATH := api/proto
API_OUT_PATH := pkg/api

setup_dev: ## start development databases
	cd deployments/docker-compose && docker-compose up -d

teardown_dev: ## stop development databases
	cd deployments/docker-compose && docker-compose down

compile: ## Build the binary file for server
	go build -i -v -o $(SERVICE_OUT) $(SERVICE_PKG_BUILD)

run: ## run compiled binary
	./$(SERVICE_OUT)

run_app: ## go run server
	go run cmd/server/*.go

run_app_online:
	go build -i -v -o $(SERVICE_OUT) $(SERVICE_PKG_BUILD)  && ./$(SERVICE_OUT) --config-file=configs/dev.yml

docker_build: ## Create a docker image for the service
ifdef tag
	@docker build -t gidyon/$(PROJECT_NAME)-api:$(tag) .
else
	@docker build -t gidyon/$(PROJECT_NAME)-api:latest .
endif

docker_tag: ## Tag image
ifdef tag
	@docker tag gidyon/$(PROJECT_NAME)-api:$(tag) gidyon/$(PROJECT_NAME)-api:$(tag)
else
	@docker tag gidyon/$(PROJECT_NAME)-api:latest gidyon/$(PROJECT_NAME)-api:latest
endif

docker_push: ## Push image
ifdef tag
	@docker push gidyon/$(PROJECT_NAME)-api:$(tag)
else
	@docker push gidyon/$(PROJECT_NAME)-api:latest
endif

build_and_push: compile docker_build docker_tag docker_push

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


# ============================================================================== 
proto_compile_location:
	protoc -I=$(API_IN_PATH) -I=third_party --go_out=plugins=grpc:$(API_OUT_PATH)/location locationv2.proto
	# protoc -I=$(API_IN_PATH) -I=third_party --grpc-gateway_out=logtostderr=true:$(API_OUT_PATH)/location locationv2.proto
	# protoc -I=$(API_IN_PATH) -I=third_party --swagger_out=logtostderr=true:$(SWAGGER_DOC_OUT_PATH) locationv2.proto