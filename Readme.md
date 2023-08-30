#### todo list
```java
0 完成 orderstates logic, gradually replace previous strategy state, remember to add some lock logic before replacement
1 任何一个 live时间久了，超过 1h都 cancel，并考虑综合
2 place order时，具体忍让的千分之点差 与 diff成一定比例，如6的话可以让千分之六
3 trigger后，两分钟内持续打印改 execSymbol的 diff及价格，考虑真实机会是不是瞬时的


```
