#### CEX option
```java
1 Please choose any top 10 CEX that support your country KYC. In this project Okex and Kucoin is selected.
```
#### Configuration
```java
Thanks to the security reasons，all sensitive config info is hardcoded in go file instead of common yaml or properties file.
- All necessary apikey/secret/passphrase of CEX can be filled in cnf.go under respective folder, i.e. /cex/kucoin
- For alarm/notify purpose, feishu webhook address can be filled in feishu.go. You can replace feishu with discord if you wish. 
- database(Mysql) config info can be filled in dbClient function within backendserver.go.
```

#### Strategy Description
```java
    Cross-CEX based arbitrage is a prevelant trade strategy that can also be applied in crypto world. In a nutshell, trader
can make profit by buying an asset(token i.e.) in one market where the asset is underpriced and simultaneously selling 
it in another market where the asset is overpriced. 
    You can make profit by adopting this strategy, but only theoretically. 
Due to the highly competitive nature of the market, opportunity is fleeting. In most cases, the market orders you placed 
can hardly be execuetd at the expected price or the limit orders could never be filled. As a result, we need to take a lot 
into consideration in the implementation of this arbitrage strategy.
```