set -euo pipefail

for CLUSTER in cluster_1 cluster_2; do
  rm -f kubeconfig-$CLUSTER
  kind delete cluster --name $CLUSTER # if exists
done
