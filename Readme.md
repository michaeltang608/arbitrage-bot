#### 交易所选择
```java
1 因为各种原因目前只对接了两家 CEX，oke 和 kucoin, 如binance不支持US服务器，huobi不接受大陆用户注册
```
#### 配置问题
```java
1 考虑开发速度以及go项目代码打包后的隐私性，配置均采用硬编码方式
- 交易所的apikey等信息均可以写在各自的 cnf.go文件中
- 告警的飞书webhook地址配置在 feishu.go中
- db信息配置在 backendserver.go 的dbClient 函数中
```