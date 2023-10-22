#!/usr/bin/env bash
# docker login
gcloud auth print-access-token | docker login \
  -u oauth2accesstoken \
  --password-stdin https://asia-southeast1-docker.pkg.dev

# docker build
docker buildx build --platform linux/amd64 -t cloudrun-conn-test .

# docker tag
docker tag cloudrun-conn-test \
  asia-southeast1-docker.pkg.dev/tdshop-data-internal/gcf-artifacts/cloudrun-conn-test:v0.1.0

# docker push
docker push asia-southeast1-docker.pkg.dev/tdshop-data-internal/gcf-artifacts/cloudrun-conn-test
