set -euo pipefail

VERSION="$1"

IMAGES=(
  "service-account-import-admission-controller"
  "service-account-import-controller"
  "multicluster-service-account-example-multicluster-client"
)

for IMAGE in "${IMAGES[@]}"; do
  kind load docker-image "quay.io/admiralty/$IMAGE:$VERSION" --name cluster_1
done
KUBECONFIG=kubeconfig-cluster_1 kubectl apply -f _out/install.yaml

OS=linux
kubemcsa="_out/kubemcsa-$OS-amd64"
$kubemcsa bootstrap \
  --target-kubeconfig kubeconfig-cluster_1 --target-name cluster-1 \
  --source-kubeconfig kubeconfig-cluster_2 --source-name cluster-2
