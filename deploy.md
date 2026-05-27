# 公网部署方案

## 方案 A：Cloudflare Tunnel（免费，需电脑开机）

1. 下载 cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/
2. 运行：
   ```bash
   cloudflared tunnel --url http://localhost:8080
   ```
3. 得到一个公网地址，如 `https://xxx.trycloudflare.com`
4. 客户通过这个地址访问激活页，自动获取 Key

优点：免费，数据在你电脑上  
缺点：你的电脑必须保持开机

## 方案 B：Railway 部署（免费额度，无需开机）

1. Fork 本仓库
2. 在 railway.app 新建项目，选择 Deploy from GitHub
3. 设置启动命令：`./wechat-obsidian-bot --serve 8080`
4. 得到 `https://xxx.railway.app` 公网地址

优点：24/7 在线，无需自己维护  
缺点：免费额度有限（每月 $5）

## 方案 C：自购 VPS（¥50/月，完全控制）

```bash
# 在 VPS 上
scp wechat-obsidian-bot.exe user@vps:/opt/bot/
ssh user@vps "./wechat-obsidian-bot --serve 8080 &"
# 配置 Nginx 反代
```

## 推荐起步方案

先用 A（Cloudflare Tunnel），零成本验证。有稳定客户后迁移到 B 或 C。
