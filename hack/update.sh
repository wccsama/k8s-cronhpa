set -o errexit
set -o nounset
set -o pipefail

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
#
 
#run "go mod vendor && chmod -R 777 vendor"
#run "cd hack && ./update-codegen.sh" in k8s-cronhpa
bash ../vendor/k8s.io/code-generator/generate-groups.sh all \
github.com/k8s-cronhpa/pkg/client \
github.com/k8s-cronhpa/pkg/api \
cronhpa:v1 \
--output-base $(pwd)../../../../ \
--go-header-file $(pwd)/boilerplate.go.txt