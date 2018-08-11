Snowflake-go
-----

基于 [bwmarrin/snowflake](https://github.com/bwmarrin/snowflake.git) 实现```grpc```,```http```,```redis```API。

使用场景: 你已经厌倦了数据库自增ID，想换一个专门的ID生成器,那么你可以试试这个.

Twitter雪花算法

```
+--------------------------------------------------------------------------+
| 1 Bit Unused | 41 Bit Timestamp |  10 Bit NodeID  |   12 Bit Sequence ID |
+--------------------------------------------------------------------------+
```

支持协议
----

#### redis

```
协议
> *2\r\n$4\r\nsfid\r\n$1\r\n5\r\n
< +{"id":"1023510546613293056","base32":"hpb7pppcywyy","base58":"3nNeLuv1t79"}

redis-cli
127.0.0.1:8199> sfid 5
{"id":"1023510546613293056","base32":"hpb7pppcywyy","base58":"3nNeLuv1t79"}
```

#### http

```
request: GET http://127.0.0.1:8199/sf-gen?node_id=2
ok response: Content-Type: application/json
{
  "status": 0,
  "message": "ok",
  "data": {
    "id": "1023526277203632128",
    "base32": "hpnmz8ioyeyy",
    "base58": "3nNmTH2pKnJ"
  }
}
error response: Content-Type: application/json
{
  "status": -1,
  "message": "error node_id"
}

```
#### grpc
    你问我支持不支持，我当然是支持的。
    (暂时没有编译客户端.)


License
-----

Apache License 2.0