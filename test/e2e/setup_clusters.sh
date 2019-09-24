set -euo pipefail

for CLUSTER in cluster1 cluster2; do
  kind create cluster --name $CLUSTER --wait 5m
  kind get kubeconfig --name $CLUSTER --internal >kubeconfig-$CLUSTER
  KUBECONFIG=kubeconfig-$CLUSTER kubectl apply -f test/e2e/must-run-as-non-root.yaml
done
