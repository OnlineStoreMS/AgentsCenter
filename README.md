# AgentsCenter

WindowsAgent 执行中心：Agent 注册 / 心跳 / 店铺会话上报 / 任务匹配下发。

## 架构

```
业务中心(售后/订单/商品)  --提交 jobType+店铺+参数-->  AgentsCenter
WindowsAgent             --注册/心跳/上报店铺------>  AgentsCenter
WindowsAgent             <--claim 任务 / report----  AgentsCenter
```

业务中心不关心哪台电脑；中心按「在线 Agent + 已登录店铺」匹配分配。

售后抓取的**业务落库**仍在 AfterSalesCore（`/plugin/sync`）；AgentsCenter 只负责任务编排。

## 端口

| Web | API | 公网 |
|-----|-----|------|
| 5192 | 8107 | `/apps/agents/` |

## 本机开发

```bash
cp configs/config.example.yaml configs/config.yaml
go run ./cmd/api -config configs/config.yaml

cd web && npm i && npm run dev
```

## Agent 机器 API

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/agent/register` | 无 |
| POST | `/api/v1/agent/heartbeat` | `X-Agent-Key` / `X-Agent-Secret` |
| GET | `/api/v1/agent/jobs/claim` | 同上 |
| POST | `/api/v1/agent/jobs/:id/report` | 同上 |

## 部署（ACR）

标准 OSMS 应用：GitHub Actions 推送 `agentscenter-api` / `agentscenter-web` 到阿里云 ACR，服务器用 deploy 仓库拉取启动。

1. 仓库：`OnlineStoreMS/AgentsCenter`（组织 Secrets：`ALIYUN_ACR_*`）
2. 推送 `main` / `dev_yeyazhou` 或手动跑 workflow `Docker Build & Push (Aliyun ACR)`
3. 服务器：`make init-external-db && make sync-configs && make pull-images && make up-images-caddy`（或现有 ACR 流程）
4. 门户路径：`https://<domain>/apps/agents/`

详见 deploy 仓库 [docs/GITHUB_ACR.md](../deploy/docs/GITHUB_ACR.md)。

## 管理端 API（JWT）

| Method | Path |
|--------|------|
| GET | `/api/v1/admin/agents` |
| GET | `/api/v1/admin/shops` |
| GET/POST | `/api/v1/admin/jobs` |
| GET | `/api/v1/admin/skills` |

## 支持的任务类型

- `doudian.aftersale`
- `doudian.order.decrypt-phone`
- `kdzs.remote.print`
