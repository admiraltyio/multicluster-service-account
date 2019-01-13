set -euo pipefail

RELEASE="$1"

sed "s/RELEASE/$RELEASE/g" release/kustomization.tmpl.yaml > release/kustomization.yaml
kustomize build release/ -o install.yaml
# TODO: upload to GitHub
RELEASE=$RELEASE skaffold build -f release/skaffold.yaml
# TODO: also tag images with latest
