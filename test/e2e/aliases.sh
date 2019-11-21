k1() { KUBECONFIG=kubeconfig-cluster_1 kubectl "$@"; }
k2() { KUBECONFIG=kubeconfig-cluster_2 kubectl "$@"; }
