# 链安智御 · 前端（组员 B）

面向银行大模型生产应用的全链路安全管控平台 —— 前端可视化层。

技术栈：**Vue 3 + Element Plus + ECharts + Pinia + Vue Router + Axios + Vite**

## 目录结构

```
frontend/
├── index.html
├── vite.config.js            # dev 代理：/api → Go 后端 127.0.0.1:8080
├── .env.development          # VITE_USE_MOCK=true 时走前端 Mock，可独立演示
└── src/
    ├── main.js               # 入口，注册 Element Plus + 图标
    ├── App.vue
    ├── router/index.js       # 路由：登录 / 态势大屏 / 攻防测试 / 审计对账
    ├── store/user.js         # 用户态（Pinia）
    ├── api/                  # 接口层（统一 axios + Mock 开关）
    │   ├── request.js        # axios 实例：拦截器 / 统一响应 / 401
    │   ├── auth.js           # 登录（组员 B 后台）
    │   ├── gateway.js        # 攻防测试（经网关 → 组员 A Redis + 组员 C AI）
    │   ├── audit.js          # 审计拓扑/告警（组员 A）
    │   ├── stats.js          # 态势统计（Go 网关）
    │   └── admin.js          # 用户/角色/数据分级 CRUD（组员 B 后台）
    ├── mock/                 # 前端 Mock 数据（后端未就绪时）
    ├── layout/MainLayout.vue
    ├── components/           # BaseChart / StatCard / PipelineStages / HashTopology
    └── views/                # Login / SecurityDashboard / AttackDefense / AuditTopology
```

## 快速开始

```bash
cd frontend
npm install
npm run dev
```

打开 http://localhost:5173 （默认跳转登录页，Mock 模式任意账号密码可登录）。

- **Mock 模式**（默认，`VITE_USE_MOCK=true`）：三个页面用本地假数据独立演示，无需任何后端。
- **联调模式**：把 `.env.development` 的 `VITE_USE_MOCK` 改为 `false`，并将 Go 后端网关跑在 `127.0.0.1:8080`。

## 与组员 A / C 的接口契约

后端统一响应格式：`{ code: 0, data: {...}, msg: 'ok' }`，`code === 0` 为成功。
所有接口前缀为 `/api`（前端已通过 `vite proxy` 与 `request.js` 的 baseURL 处理）。

| 页面 | 方法 / 路径 | 提供方 | 请求 | 响应 data |
| --- | --- | --- | --- | --- |
| 登录 | `POST /api/auth/login` | 组员 B 后台 | `{ username, password }` | `{ token, user:{username,role,dataLevel} }` |
| 攻防测试（注入） | `POST /api/gateway/attack-test` | 组员 B 网关（转发组员 C + 组员 A） | `{ prompt }` | `{ requestId, prompt, verdict, totalLatencyMs, stages[] }` |
| 攻防测试（越权） | `POST /api/gateway/access-test` | 同上 | `{ role, dataLevel, action }` | 同上 |
| 态势总览 | `GET /api/stats/overview` | 组员 B 网关 | - | `{ totalRequests, blockedToday, blockRate, highRiskUsers }` |
| 拦截趋势 | `GET /api/stats/trend` | 组员 B 网关 | - | `[{ time, blocked, passed }]` |
| 风险分布 | `GET /api/stats/risk-distribution` | 组员 B 网关 | - | `[{ type, value }]` |
| 高风险用户 | `GET /api/stats/high-risk-users` | 组员 B 网关 | - | `[{ name, role, riskScore, lastAction, level }]` |
| 审计拓扑 | `GET /api/audit/topology` | 组员 A | - | `{ nodes[], edges[], chainStatus, alert }` |
| 告警日志 | `GET /api/audit/alerts` | 组员 A | - | `[{ id, time, node, type, message }]` |
| 用户列表 | `GET /api/users` | 组员 B 后台 | - | `[{ id, username, role, dataLevel }]` |
| 用户详情 | `GET /api/users/:id` | 组员 B 后台 | - | `{ id, username, role, dataLevel }` |
| 新增用户 | `POST /api/users` | 组员 B 后台 | `{ username, role, dataLevel, password? }` | 创建后的用户对象（`201`） |
| 更新用户 | `PUT /api/users/:id` | 组员 B 后台 | `{ role?, dataLevel? }` | 更新后的用户对象 |
| 删除用户 | `DELETE /api/users/:id` | 组员 B 后台 | - | `{ id, deleted }` |
| 角色列表 | `GET /api/roles` | 组员 B 后台 | - | `[{ name, maxAccessLevel, chainRoleOrdinal }]` |
| 数据分级 | `GET /api/data-levels` | 组员 B 后台 | - | `[{ level, desc, fields, rank }]` |

> 后端共暴露 28 条路由（`/api/v1` 12 条 + `/api` 16 条）。完整字段说明、业务校验规则与错误码
> 见 [docs/API接口规范文档.md](../docs/API接口规范文档.md)。

### 业务后台接口的校验规则

以下为后端已实现的业务校验，前端可据此做前置提示（但**以服务端校验为准**）：

| 场景 | 结果 |
| --- | --- |
| 新增用户：用户名为空 / 角色非法 / 密级非法 | `400` |
| 新增用户：**密级超出该角色可访问上限**（如"普通柜员"配 L4） | `400` |
| 新增用户：用户名已存在 | `409` |
| 更新用户：`role` 与 `dataLevel` 均未提供 | `400` |
| 更新用户：更新后的角色-密级组合越权 | `400` |
| 更新/删除用户：用户不存在 | `404` |
| 删除用户：内置 `admin` 账号 | `403` |

**角色可访问最高密级**：管理员 L4 · 风控审核员 L3 · 普通柜员 L2 · 客服坐席 L2 · 未识别角色 L1。

### 攻防测试 `stages[]` 结构（核心契约）

```jsonc
{
  "key": "input-risk",        // auth | input-risk | access-control | infer | output-sanitize
  "name": "AI 输入风险检测",
  "owner": "C",               // B=网关 / A=链上策略 / C=AI 服务
  "status": "block",          // pass | block | skip
  "latencyMs": 12,
  "message": "检测到越狱指令…",
  "extra": { "riskScore": 92, "riskType": "jailbreak" }
}
```

> 前端只消费上述字段；组员 A / C 按此结构返回即可直接对接，前端无需改动。

### 审计拓扑 `topology` 结构

```jsonc
{
  "nodes": [{ "id": "rag", "name": "RAG 检索节点", "hash": "0x...", "status": "tampered", "x": 300, "y": 200 }],
  "edges": [{ "from": "access", "to": "rag" }],
  "chainStatus": "inconsistent",   // consistent | inconsistent
  "alert": "RAG 检索节点 Hash 与链上记录不一致"
}
```

`status: 'tampered'` 的节点前端会自动加红色粗边框标记。

## 构建

```bash
npm run build      # 产物在 dist/
npm run preview    # 预览构建产物
```
