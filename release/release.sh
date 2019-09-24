set -euo pipefail

VERSION="$1"

IMAGES=(
  "service-account-import-admission-controller"
  "service-account-import-controller"
  "multicluster-service-account-example-generic-client"
  "multicluster-service-account-example-multicluster-client"
)

for IMAGE in "${IMAGES[@]}"; do
  docker push "quay.io/admiralty/$IMAGE:$VERSION"
done

# TODO: upload install.yaml and kubemcsa binaries to GitHub
# TODO: also tag images with latest
