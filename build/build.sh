set -euo pipefail

VERSION="$1"

ROOT_PKG="admiralty.io/multicluster-service-account"
TARGETS=(
  "cmd/service-account-import-admission-controller"
  "cmd/service-account-import-controller"
  "examples/generic-client"
  "examples/multicluster-client"
)
IMAGES=(
  "service-account-import-admission-controller"
  "service-account-import-controller"
  "multicluster-service-account-example-generic-client"
  "multicluster-service-account-example-multicluster-client"
)

for TARGET in "${TARGETS[@]}"; do
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "_out/$TARGET/manager" "$ROOT_PKG/$TARGET"
done

for OS in linux darwin windows; do
  CGO_ENABLED=0 GOOS=$OS GOARCH=amd64 go build -o "_out/kubemcsa-$OS-amd64" "$ROOT_PKG/cmd/kubemcsa"
done

cp build/Dockerfile _out/

for ((i = 0; i < ${#TARGETS[@]}; ++i)); do
  docker build -t "quay.io/admiralty/${IMAGES[i]}:$VERSION" --build-arg target="${TARGETS[i]}" _out
done

cat config/crds/*.yaml config/install/*.yaml | sed "s/MY_VERSION/$VERSION/g" >_out/install.yaml
