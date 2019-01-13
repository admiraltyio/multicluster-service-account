set -euo pipefail

kubectl config use-context cluster1 && skaffold run -f test/e2e/cluster1/skaffold.yaml
sleep 5 # fix race condition
./kubemcsa-darwin-amd64 bootstrap cluster1 cluster2 # TODO: accept OS as parameter
kustomize build test/e2e/cluster2/ | kubectl --context cluster2 apply -f -
kustomize build test/e2e/cluster1/ | kubectl --context cluster1 apply -f -
