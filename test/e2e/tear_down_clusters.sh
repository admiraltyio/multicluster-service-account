set -euo pipefail

for CLUSTER in cluster1 cluster2; do
  rm -f kubeconfig-$CLUSTER
  kind delete cluster --name $CLUSTER # if exists
done
