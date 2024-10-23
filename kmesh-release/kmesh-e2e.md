
以下命令不会重新安装Kind集群及Kmesh，并且在测试结束后不清环境
```bash
go test -v -tags=integ ./test/e2e/... --istio.test.kube.deploy=false --istio.test.nocleanup
```