# Build the manager binary
FROM quay.io/operator-framework/helm-operator:v1.22.2

ADD LICENSE /licenses/LICENSE

LABEL name="apisix-operator" \
	maintainer="cheverjonathan@gmail.com" \
	version="v0.0.1"
	summary="apisix-operator installs and manages your Apache APISIX in k8s environment" 

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts  ${HOME}/helm-charts
WORKDIR ${HOME}
