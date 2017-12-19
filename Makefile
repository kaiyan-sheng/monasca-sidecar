NAME=monasca-sidecar

default: $(NAME)

$(NAME):
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/$(NAME) .

fast:
	go build -o bin/$(NAME) .

depend:
	glide install

test: $(NAME)
	@echo -e "\nRunning all go tests:"
	@echo -e "------------------------------------------------------------------------"
    export NOLOGGING=true; go test $$(go list ./... | grep -v /vendor/)

clean:
	rm -rf ./vendor
	rm ./bin/$(NAME)
