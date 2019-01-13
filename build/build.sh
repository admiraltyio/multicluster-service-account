set -euo pipefail

ROOT_PKG="admiralty.io/multicluster-service-account"
TARGETS=(
	"cmd/service-account-import-admission-controller"
	"cmd/service-account-import-controller"
	"examples/generic-client"
	"examples/multicluster-client"
)

for TARGET in "${TARGETS[@]}"; do
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "_out/$TARGET/manager" "$ROOT_PKG/$TARGET"
done

for OS in linux darwin windows
do
	CGO_ENABLED=0 GOOS=$OS GOARCH=amd64 go build -o "_out/kubemcsa-$OS-amd64" "$ROOT_PKG/cmd/kubemcsa"
done
