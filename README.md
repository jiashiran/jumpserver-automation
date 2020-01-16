# jumpserver-automation
自动化登录jumperserver执行命令

# docker 形式部署

## build
```
cd build
./build.sh
```

## run
```
docker run -d --name jumpserver-auto --net=host --restart=always -v /usr/local/db/:/usr/local/db/ 7a9eb58432dd
```
