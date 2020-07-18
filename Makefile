GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOBASE := $(shell pwd)
GOBIN=$(GOBASE)/bin
BINARY_NAME=authservice
    
all: test build
build: 
	$(GOBUILD) -o $(GOBIN)/$(BINARY_NAME) -v
test:
	$(GOTEST) ./...
clean:
	$(GOCLEAN)
	rm -f $(GOBIN)/$(BINARY_NAME)
run:
	$(GOBUILD) -o $(BINARY_NAME) ./...
	./$(BINARY_NAME)
    