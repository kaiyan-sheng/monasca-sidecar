NAME=monasca-sidecar

default: $(NAME)

$(NAME):
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/$(NAME) .

fast:
	go build -o bin/$(NAME) .

depend:
	glide install

clean:
	rm -rf ./vendor
	rm ./bin/$(NAME)
