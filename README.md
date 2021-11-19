# SERVER-TAOJAN
> 服务端版本请使用 [xflash-panda/v2board](https://github.com/xflash-panda/v2board), 不要使用原版

## 主要特性
- 永久免费，并且完整开源
- 无需配置文件，和面板完美集成
- 专属面板，专属协议，更简单的实现方式，
- 更好的性能，更低的资源占用，优化大量无效数据传输，降低服务端负载
- 陆续会有新特性支持

## 安装
**手动安装**
1. go >= 1.16.0
2. 依次运行
```
git clone https://github.com/xflash-panda/server-trojan.git
cd server-trojan/cmd
go build -o server-trojan -ldflags "-s -w"
chmod +x server-trojan
./server-trojan --api xxx --token xxx --node xxx
```
**一键安装**
* [server-trojan-install](https://github.com/xflash-panda/server-trojan-install)


##  Thanks
* [Project X](https://github.com/XTLS/)
* [XrayR](https://github.com/XrayR-project/XrayR)