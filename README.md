# aliddns

- 下载[aliddns](https://github.com/Kycer/aliddns/releases/tag/0.0.1)
- 创建配置文件
```shell
mkdir ~/.config/aliddns
cd ~/.config/aliddns
touch 域名.toml # touch github.com.toml
```
- 配置文件示例
    ```shell
    [aliaccess]
    accessid = "xxxxxxxxxxxxxxxxxxxxx"
    accesskey = "xxxxxxxxxxxxxxxxxxxxx"
    region = "cn-hangzhou"

    [[domains]]
    domainType = "A"
    rr = "www"
    updateType = "network"

    [[domains]]
    domainType = "A"
    rr = "@"
    updateType = "network"

    [[domains]]
    domainType = "A"
    rr = "test"
    updateType = "network"

    [[domains]]
    domainType = "A"
    rr = "local"
    updateType = "local"
    value = "127.0.0.1"
    ```
- 启动
```shell
./aliddns  # 执行一次
./aliddns -o=false -e=20  # 每20分钟一次
nohup ./aliddns -o=false -e=1 & # 后台启动
```