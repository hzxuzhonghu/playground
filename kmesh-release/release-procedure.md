### kmesh release procedure

1. cut a new release branch `release-x.y`
1. update the VERSION in makefile to `x.y.z`
1. update tag in helm chart values.yaml to `vx.y.z`
1. make helm-package CHART_VERSION=v0.5.0-rc.0
1. make helm-push CHART_VERSION=v0.5.0-rc.0
