#!/bin/bash


mode=${1:-}
qps=${2:0}
namespace=test
client_address=deployment/fortio-client
server_address=http://fortio-server.$namespace:8080

time=$(date "+%Y%m%d%H%M%S")

log_path=long_connection/${mode}/$time
mkdir -p ${log_path}

for theadnum in 1 2 4 8 16 32 64 128; do
        echo "run $theadnum..."
        echo "kubectl exec -it ${client_address} -n $namespace -- fortio load -quiet -c ${theadnum} -t 30s -keepalive=true -qps ${qps} ${server_address}" > ${log_path}/test_${theadnum}.log
        # dstat -cmtn 5s >> ${log_path}/test_${theadnum}.log &
        kubectl exec -it ${client_address} -n $namespace -- fortio load -quiet -c ${theadnum} -t 30s -keepalive=true -qps ${qps} ${server_address} >> ${log_path}/test_${theadnum}.log
        sleep 10
        # ps -ef | grep dstat | grep -v grep | awk {'print $2'} | xargs kill
done
