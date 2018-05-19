#!/bin/bash

# Docker registry & docker tag
echo "STABLE_DOCKER_TAG ${DOCKER_TAG:-latest}"
USER=$(whoami)
echo "STABLE_DOCKER_REGISTRY ${DOCKER_REGISTRY:-$USER}"


