PROJECT_NAME := pandemic-api
PKG := gtuhub.com/gidyon/$(PROJECT_NAME)

compile:
	go build -i -v -o restful .

copy_dir:
	cp -r /home/gideon/go/src/github.com/gidyon/pandemic-api/api/json .

docker_build:
ifdef tag
	@docker build -t gidyon/$(PROJECT_NAME)-restful:$(tag) .
else
	@docker build -t gidyon/$(PROJECT_NAME)-restful:latest .
endif

rm_dir:
	rm -rf /home/gideon/go/src/github.com/gidyon/pandemic-api/cmd/restful/json

docker_tag:
ifdef tag
	@docker tag gidyon/$(PROJECT_NAME)-restful:$(tag) gidyon/$(PROJECT_NAME)-restful:$(tag)
else
	@docker tag gidyon/$(PROJECT_NAME)-restful:latest gidyon/$(PROJECT_NAME)-restful:latest
endif

docker_push:
ifdef tag
	@docker push gidyon/$(PROJECT_NAME)-restful:$(tag)
else
	@docker push gidyon/$(PROJECT_NAME)-restful:latest
endif

build_image: docker_build docker_tag docker_push

build: compile copy_dir docker_build docker_tag docker_push rm_dir