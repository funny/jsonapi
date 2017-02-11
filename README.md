介绍
====

这是一个简易的Web接口框架，有以下特性：

1. 请求和响应纯粹只用JSON格式
2. 统一的防回放攻击机制
3. 无第三方库依赖
4. 速错原则

用法
====

示例：

```go
api := New(crypto.SHA256, jsonapi.StdLogger)

api.HandleFunc("/api", func(ctx *jsonapi.Context) interface{} {
	var req map[string]interface{}

	ctx.Request(&req)

	return map[string]interface{}{
		"value_is": req["value"],
	}
})

go http.ListenAndServe(":8080", api)
```

防回放机制
========

防回放机制支持有效期验证和请求参数签名验证。

这套机制的基本算法是：hash(key + time + path + json)

其中hash算法可以在App实例化时指定；time为客户端通过HTTP头告诉服务端的请求签名时间；path为当前请求的路径；json为请求内容。

客户端通过HTTP头`t`告知服务端请求签名时间，这个时间为unix时间（UTC时区）；通过HTTP头`s`告知服务端请求签名。

服务端通过调用`ctx.Verify(key, timeout)`来验证请求有消息，此调用必须在`ctx.Request()`调用之后。

传入`Verify`方法的`key`为空字符串时，框架不验证请求签名。

传入`Verify`方法的`timeout`为0时，框架不验证请求有效期。

更具体的验证逻辑请阅读`Verify`方法的逻辑。
