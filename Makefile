.PHONY: gazelle
gazelle:
	bazel run //:gazelle -- -proto disable

.PHONY: goimports
goimports:
	goimports -w cmd/ pkg/

