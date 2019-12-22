install:
	go get

build:
	go build -i -ldflags="-X github.com/hpcsc/aws-profile-utils/handlers.version=$(version)" -o bin/aws-profile-utils github.com/hpcsc/aws-profile-utils
	cat bintray-descriptor.json | sed -E 's/AWS_PROFILE_UTILS_VERSION/'${version}'/' | tee bintray-descriptor.json

test:
	go test -v ./...

run:
	./bin/aws-profile-utils $(args)

all: build run