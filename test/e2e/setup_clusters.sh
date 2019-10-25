set -euo pipefail

for CLUSTER in cluster_1 cluster_2; do # using underscores in cluster names to test kubemcsa bootstrap name overrides
  kind create cluster --name $CLUSTER --wait 5m
  kind get kubeconfig --name $CLUSTER --internal >kubeconfig-$CLUSTER
  KUBECONFIG=kubeconfig-$CLUSTER kubectl apply -f test/e2e/must-run-as-non-root.yaml
done
