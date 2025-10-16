# Docker Performance Monitor – Web Dashboard

This SvelteKit app was scaffolded with `npx sv create web` and extended with TailwindCSS and shadcn-svelte components. It provides the browser dashboard shell for the Docker Performance Monitor project.

## Prerequisites

- Node.js 22 (see `.nvmrc`)
- `pnpm` (Corepack users can run `corepack enable pnpm`)

## Install & Run

```bash
pnpm install
pnpm dev
```

The development server runs on `http://localhost:5173`. A mock WebSocket feed powers the dashboard until the Go agent integration is completed.

## (Refactored) SSE Aggregation Mode

Instead of the browser opening many WebSocket connections directly to each Docker stats agent, the SvelteKit server now acts as an in-process aggregator:

1. The server (Node runtime) opens and maintains internal WebSocket connections to all configured agent endpoints (`ws://...`).
2. The browser subscribes to a single **Server-Sent Events (SSE)** endpoint: `/stream/agents`.
3. All agent status changes and container stats batches are multiplexed and streamed down that one channel.

### Environment Variable

Create `.env.local` in the `web/` directory (see `.env.local.example`):

```
AGENT_ENDPOINTS=ws://10.33.1.166:8080/ws;ws://10.33.1.167:8080/ws
```

Accepted formats (semicolon separated):
1. Simple: `ws://host1:8080/ws;ws://host2:8080/ws`
2. Detailed: `id|label|ws://host:port/ws;id2|label2|ws://...`

Example:
```
AGENT_ENDPOINTS=agent-a|Host A|ws://10.33.1.166:8080/ws;agent-b|DB Node|ws://10.33.1.167:8080/ws
```

### Run in Dev

```bash
pnpm install
pnpm dev
```

The UI automatically receives agent data; no manual add/remove UI is needed anymore.

### Production Notes
`@sveltejs/adapter-auto` may not guarantee a stable long‑running process for persistent SSE. Prefer:
```
pnpm add -D @sveltejs/adapter-node
```
`svelte.config.js`:
```js
import adapter from '@sveltejs/adapter-node';
export default { kit: { adapter: adapter() } };
```
Then build & start:
```bash
pnpm build
node build/index.js
```

If you put Nginx / Caddy / Traefik in front, just proxy `/stream/agents` as a normal HTTP request keeping the `Content-Type: text/event-stream` header—no WebSocket upgrade required.

### SSE Event Types
| type | description |
|------|-------------|
| `agent_list` | Initial full list of configured agents |
| `agent_status` | Connection lifecycle changes (connecting / connected / error / closed) |
| `container_stats_batch` | Raw container stats batch forwarded from an agent |

### FAQ
| Problem | Cause | Fix |
|---------|-------|-----|
| No data | Server cannot reach agent WS | Verify `AGENT_ENDPOINTS` and network routing from server host |
| Drops / no reconnect | Intermediate proxy timeout | SSE auto‑reconnect; tune proxy timeouts / send heartbeats (already implemented) |
| Scale horizontally | In-memory hub not shared | Externalize aggregator (separate service) or use a pub/sub bus (Redis / NATS) |

### Future Enhancements
* Rolling in-memory history buffer (N minutes) for mini trend charts
* Auth token check on SSE endpoint
* Subscription filtering via query params (e.g. `?agents=agent-a,agent-b`)
* Optional WebSocket frontend for bi-directional control (pause / filter / adjust cadence)

---

## （已重构）SSE 聚合模式

前端不再直接连接多个 Agent 的 WebSocket，而是通过服务端（SvelteKit 内部 Node 运行时）建立到内网 Agents 的 WebSocket 长连接，并向浏览器提供统一的 **SSE (Server-Sent Events)** 流 `/stream/agents`。

### 环境变量配置

在项目根目录 `web/` 下创建 `.env.local`（可参考 `.env.local.example`）：

```
AGENT_ENDPOINTS=ws://10.33.1.166:8080/ws;ws://10.33.1.167:8080/ws
```

支持两种格式：
1. 简单：`ws://host1:8080/ws;ws://host2:8080/ws`
2. 详细：`id|label|ws://host:port/ws;id2|label2|ws://...`

示例：
```
AGENT_ENDPOINTS=agent-a|宿主机A|ws://10.33.1.166:8080/ws;agent-b|数据库节点|ws://10.33.1.167:8080/ws
```

### 运行

```bash
pnpm install
pnpm dev
```

浏览器访问后会自动通过 SSE 订阅聚合数据，无需在界面手动添加 / 删除 Agent。

### 生产部署注意
`@sveltejs/adapter-auto` 不能保证在所有平台稳定保持长连接。推荐改用：
```
pnpm add -D @sveltejs/adapter-node
```
修改 `svelte.config.js`：
```js
import adapter from '@sveltejs/adapter-node';
export default { kit: { adapter: adapter() } };
```
启动：
```bash
node build/index.js
```

前置 Nginx / Caddy / Traefik 时，只需把 `/stream/agents` 按普通 HTTP 代理（保持 `text/event-stream` 头）即可；无需 WebSocket 升级。

### SSE 事件类型
| type | 描述 |
|------|------|
| `agent_list` | 初始 Agent 列表 |
| `agent_status` | 连接状态变化（connecting / connected / error / closed） |
| `container_stats_batch` | 透传每次批量容器统计（含原始 payload） |

### 常见问题
| 问题 | 说明 | 解决 |
|------|------|------|
| 没有数据 | 内网 ws 访问不到 | 确认 AGENT_ENDPOINTS 正确、服务器能直连内网 IP |
| 浏览器断开后不恢复 | SSE 网络中断 | 浏览器会自动重连；可查看控制台是否被反向代理超时裁剪 |
| 多实例水平扩展 | 内存状态不共享 | 抽出聚合层到独立服务或引入消息总线（Redis / NATS） |

### 后续可扩展
* 增加历史窗口缓存（最近 N 分钟）
* 鉴权（SSE 请求 Header 里携带 Token）
* 订阅过滤（URL 查询参数选择部分 Agent）
* SSE -> WebSocket 升级以支持前端下发指令

---

中文快速指南：
1. `.env.local` 写好 `AGENT_ENDPOINTS`
2. `pnpm dev` 启动
3. 浏览器打开后即显示各 Agent 指标，无需手动添加
4. 生产部署换成 `adapter-node` + 反向代理
5. 需要公网访问：只暴露 SvelteKit 端口（或其反代），内网 Agents 不对公网开放


## Available Scripts

| Command            | Description                                       |
| ------------------ | ------------------------------------------------- |
| `pnpm dev`         | Start the Vite dev server                         |
| `pnpm build`       | Generate a production build                       |
| `pnpm preview`     | Preview the production build                      |
| `pnpm lint`        | Run Prettier and ESLint                           |
| `pnpm check`       | Run Svelte type checks                            |
| `pnpm test`        | Execute Vitest unit tests                         |
| `pnpm test:e2e`    | Placeholder Playwright command (no specs yet)     |

## Component Library

UI primitives are generated via `pnpm dlx shadcn-svelte@latest init` and `pnpm dlx shadcn-svelte add ...`. Regenerate components or update the registry with the same commands.

## Re-scaffolding

If the project ever needs a fresh scaffold, run:

```bash
pnpx sv@latest create web --template minimal --types ts
```

Then re-apply the TailwindCSS add-on (`pnpm dlx sv add tailwindcss`) and re-run the shadcn initialisation step described above.

Everything you need to build a Svelte project, powered by [`sv`](https://github.com/sveltejs/cli).

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project in the current directory
npx sv create

# create a new project in my-app
npx sv create my-app
```

## Developing

Once you've created a project and installed dependencies with `npm install` (or `pnpm install` or `yarn`), start a development server:

```sh
npm run dev

# or start the server and open the app in a new browser tab
npm run dev -- --open
```

## Building

To create a production version of your app:

```sh
npm run build
```

You can preview the production build with `npm run preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
