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
