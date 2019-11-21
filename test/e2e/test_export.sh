set -euo pipefail

source test/e2e/aliases.sh

OS=linux
kubemcsa="_out/kubemcsa-$OS-amd64"
$kubemcsa export --kubeconfig kubeconfig-cluster_1 default | k2 apply -f -

NAME=$(k1 get secret -o name)
k2 get $NAME
