# 真实环境 E2E 测试

将仓库根目录 `.env.example` 复制为 `.env`，填写：

- `TUITUI_BOT_APPID`、`TUITUI_BOT_SECRET`
- `TUITUI_BOT2_APPID`、`TUITUI_BOT2_SECRET`
- `TARGET_ACCOUNT`、`TARGET_GROUP`
- `TARGET_TEAM`、`TARGET_CHANNEL`

也可以直接设置同名进程环境变量，已有环境变量优先于根 `.env`。运行：

```
go test -tags e2e ./tests/e2e
```

测试会发送真实消息、发布团队帖子并读取文件空间。双 Bot 测试会为 Bot 1 建立 WebSocket；运行前
应停止其他使用相同凭据接收事件的进程。
