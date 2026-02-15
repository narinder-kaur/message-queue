set -e
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then PLAT=darwin-arm64; else PLAT=darwin-amd64; fi
TAG=$(curl -s https://api.github.com/repos/helm/helm/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
URL="https://get.helm.sh/helm-${TAG}-${PLAT}.tar.gz"
mkdir -p .bin /tmp/helm-download
curl -fsSL "$URL" -o /tmp/helm-download/helm.tar.gz
tar -xzf /tmp/helm-download/helm.tar.gz -C /tmp/helm-download
if [ -f "/tmp/helm-download/${PLAT}/helm" ]; then mv "/tmp/helm-download/${PLAT}/helm" .bin/helm; elif [ -f "/tmp/helm-download/helm" ]; then mv /tmp/helm-download/helm .bin/helm; else echo "helm binary not found in archive"; exit 1; fi
chmod +x .bin/helm
./.bin/helm version
./.bin/helm lint charts/message-streaming-app || true
./.bin/helm template msa charts/message-streaming-app --values charts/message-streaming-app/values.yaml > /tmp/msa-render.yaml
ls -l /tmp/msa-render.yaml

# curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
# helm lint charts/message-streaming-app
# helm template msa charts/message-streaming-app --values charts/message-streaming-app/values.yaml > /tmp/msa-render.yaml
# kubectl apply --dry-run=client -f /tmp/msa-render.yaml