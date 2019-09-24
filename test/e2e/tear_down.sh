set -euo pipefail

KUBECONFIG=kubeconfig-cluster1 kubectl delete -f _out/install.yaml
KUBECONFIG=kubeconfig-cluster2 kubectl delete namespace multicluster-service-account # TODO? run kubemcsa unstrap when it exists
