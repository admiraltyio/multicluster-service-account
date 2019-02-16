set -euo pipefail

kubectl config use-context cluster1 && skaffold run -f test/e2e/cluster1/skaffold.yaml
_out/kubemcsa-darwin-amd64 bootstrap cluster1 cluster2 # TODO: accept OS as parameter
