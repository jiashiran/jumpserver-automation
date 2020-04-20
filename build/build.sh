#!/usr/bin/env bash

cd ../

export GOOS=linux
go build -o app
chmod +x app
mv app build/
cp -r static build/
cd build

docker build -t registry.cn-beijing.aliyuncs.com/tinet-dev/jumpserver-autu .
rm -rf app
rm -rf static
docker push registry.cn-beijing.aliyuncs.com/tinet-dev/jumpserver-autu

