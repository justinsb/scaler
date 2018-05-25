DOCKER_REGISTRY?=index.docker.io
DOCKER_IMAGE_PREFIX?=$(shell whoami)/
DOCKER_TAG?=latest

.PHONY: scale
scaler:
	bazel build //cmd/scaler

.PHONY: gazelle
gazelle:
	bazel run //:gazelle -- -proto disable

.PHONY: goimports
goimports:
	goimports -w cmd/ pkg/

.PHONY: dep
dep-ensure:
	dep ensure
	find vendor/ -name "BUILD" -delete

.PHONY: dep
dep: dep-ensure gazelle
	@echo "Updated deps and ran gazelle"

.PHONY: test
test:
	bazel test //cmd/... //pkg/...

.PHONY: push
push:
	bazel run //images:push-scaler

.PHONY: images
images:
	bazel run //images:scaler
	docker tag bazel/images:scaler ${DOCKER_IMAGE_PREFIX}scaler:${DOCKER_TAG}

bounce:
	kubectl delete pod -n kube-system -l k8s-addon=scaler
