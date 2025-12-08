IMAGE_PREFIX ?= pradyunuydarp/lastmile
TAG ?= latest
DOCKER ?= docker

.PHONY: build-all push-all

build-all:
	$(DOCKER) build -t $(IMAGE_PREFIX)-driver:$(TAG) --build-arg SERVICE=driver -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-rider:$(TAG) --build-arg SERVICE=rider -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-station:$(TAG) --build-arg SERVICE=station -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-trip:$(TAG) --build-arg SERVICE=trip -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-notification:$(TAG) --build-arg SERVICE=notification -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-location:$(TAG) --build-arg SERVICE=location -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-matching:$(TAG) --build-arg SERVICE=matching -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-user:$(TAG) --build-arg SERVICE=user -f Dockerfile .
	$(DOCKER) build -t $(IMAGE_PREFIX)-gateway:$(TAG) --build-arg SERVICE=gateway -f Dockerfile .

push-all: build-all
	$(DOCKER) push $(IMAGE_PREFIX)-driver:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-rider:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-station:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-trip:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-notification:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-location:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-matching:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-user:$(TAG)
	$(DOCKER) push $(IMAGE_PREFIX)-gateway:$(TAG)
