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
    ├── router/index.js       # 登录、业务、风控、审计、系统管理与权限说明
    ├── store/user.js         # 用户态（Pinia）
    ├── api/                  # 接口层（统一 axios + Mock 开关）
    │   ├── request.js        # axios 实例：拦截器 / 统一响应 / 401
    │   ├── auth.js           # 登录、身份恢复与退出
    │   ├── business.js       # 授权业务记录与提交
    │   ├── risk.js           # 告警风险复核
    │   ├── gateway.js        # 攻防测试（经网关 → 组员 A Redis + 组员 C AI）
    │   ├── audit.js          # 审计拓扑/告警（组员 A）
    │   ├── stats.js          # 态势统计（Go 网关）
    │   ├── admin.js          # 用户 CRUD 与只读角色/数据分级定义
    │   └── system.js         # 安全策略与运行配置
    ├── mock/                 # 前端 Mock 数据（后端未就绪时）
    ├── layout/MainLayout.vue
    ├── components/           # BaseChart / StatCard / PipelineStages / HashTopology
    └── views/                # 登录、业务工作台、风控复核、态势、审计、系统管理
```

## 快速开始

```bash
cd frontend
npm install
npm run dev
```

打开 http://localhost:5173 （默认跳转登录页，选择对应角色的演示账号登录）。

| 密级 | 账号 | 角色 | 密码 |
| --- | --- | --- | --- |
| L1 | `teller` | 柜员／客服 | `teller` |
| L3 | `reviewer` | 风控审核员 | `reviewer` |
| L2 | `auditor` | 审计人员 | `auditor` |
| L4 | `admin` | 系统管理员 | `admin` |

登录页可一键填入演示凭据。Mock 与后端均校验账号和密码；未知账号或错误密码不能登录。页面导航通过 `/api/auth/me` 恢复服务端身份，退出会撤销对应会话。旧版登录缓存会失效，需重新认证。

角色控制操作，账号数据密级与 `scope: { departments, ownerOnly }` 控制记录范围，三者分别校验。四个角色都可单独配置 L1–L4；表中是默认账号的密级。`admin` 的 L4 不授予客户数据访问权，其业务范围为空。

| 角色 | 开放页面与操作 | 默认业务范围 |
| --- | --- | --- |
| 柜员／客服 | `/business`：本人业务处理、结果与风险提示，不开放全局安全日志 | 本人发起的零售业务 |
| 风控审核员 | `/dashboard`、`/risk-review`、`/business`、`/attack-defense`：授权态势、告警详情、风险复核与演示测试 | 零售业务部、L3 及以下 |
| 审计人员 | `/audit-topology`、`/business`：授权审计记录、证据与对账，全部只读 | 零售与信贷业务、L2 及以下 |
| 系统管理员 | `/administration`：账号、角色定义、策略与运行配置 | 无客户业务范围 |

所有账号均可打开 `/my-access` 查看自身权限；未授权页面转到 `/access-denied`。

- **Mock 模式**（默认，`VITE_USE_MOCK=true`）：按已认证角色与业务范围提供本地演示数据，无需后端。
- **联调模式**：把 `.env.development` 的 `VITE_USE_MOCK` 改为 `false`，并将 Go 后端网关跑在 `127.0.0.1:8080`。

## 界面与交互

- 各角色显示对应导航与工作台，桌面使用侧栏，窄屏使用抽屉导航。
- 浅色与深色主题默认跟随系统，可在右上角切换，选择会保存在当前浏览器。
- 态势数据每 5 秒刷新，后台标签页暂停轮询；刷新失败保留上次成功的数据并显示提示，首次加载失败提供重试入口。
- 高风险用户支持搜索与风险等级筛选，告警日志支持搜索，节点 Hash 支持复制。
- 登录与攻防表单包含输入校验、等待和错误状态；未知裁决与未知节点状态不会显示为通过或正常。
- 未登录访问工作台会跳转登录，成功后回到有权访问的原页面或该角色的默认工作台。前端和后端都校验操作权限；业务记录另按密级、部门与所有者过滤。
- 风控审核员可记录“确认风险”或“排除风险”的复核结论；审计人员只能查看。复核不会解除拦截或扩大账号权限。
- 系统管理支持新增英文账号、编辑角色/密级/范围、删除账号，以及保存演示测试开关和 AI 等待时间。内置 `admin` 不可修改或删除。
- 顶栏会根据 `VITE_USE_MOCK` 显示“演示数据”或“后端联调”，登录提示也随环境变化。
- Element Plus 组件、样式、图标及 ECharts 图表按需引入；图表根据容器尺寸自动调整，并遵循减少动态效果的系统设置。

可复用的主题和加载状态位于 `src/composables/`，页面标题与导航位于 `src/components/PageHeading.vue` 和 `PlatformNav.vue`。

## 与组员 A / C 的接口契约

后端统一响应格式：`{ code: 0, data: {...}, msg: 'ok' }`，`code === 0` 为成功。
所有接口前缀为 `/api`（前端已通过 `vite proxy` 与 `request.js` 的 baseURL 处理）。

| 页面 | 方法 / 路径 | 提供方 | 请求 | 响应 data |
| --- | --- | --- | --- | --- |
| 登录 | `POST /api/auth/login` | 组员 B 后台 | `{ username, password }` | `{ token, user }` |
| 身份恢复 | `GET /api/auth/me` | Go 网关 | - | `{ user }` |
| 退出 | `POST /api/auth/logout` | Go 网关 | - | 撤销当前令牌 |
| 业务记录 | `GET /api/business/requests` | Go 网关 | - | 授权请求数组 |
| 提交业务 | `POST /api/business/requests` | Go 网关 | `{ prompt, dataLevel, businessType, action? }` | 业务结果、风险提示及请求记录 |
| 攻防测试（注入） | `POST /api/gateway/attack-test` | 组员 B 网关（转发组员 C + 组员 A） | `{ prompt }` | `{ requestId, prompt, verdict, totalLatencyMs, stages[], detection, alertIds[] }` |
| 攻防测试（越权） | `POST /api/gateway/access-test` | 同上 | `{ role, dataLevel, action }` | 同上 |
| 态势总览 | `GET /api/stats/overview` | 组员 B 网关 | - | `{ totalRequests, blockedToday, blockRate, highRiskUsers }` |
| 拦截趋势 | `GET /api/stats/trend` | 组员 B 网关 | - | `[{ time, blocked, passed }]` |
| 风险分布 | `GET /api/stats/risk-distribution` | 组员 B 网关 | - | `[{ type, value }]` |
| 高风险用户 | `GET /api/stats/high-risk-users` | 组员 B 网关 | - | `[{ name, role, riskScore, lastAction, level }]` |
| 审计拓扑 | `GET /api/audit/topology` | 组员 A | - | `{ nodes[], edges[], chainStatus, alert }` |
| 告警日志 | `GET /api/audit/alerts` | 组员 A | - | `[{ id, time, node, type, message }]` |
| 告警详情 | `GET /api/audit/alerts/:id` | Go 网关 | - | `{ id, time, node, type, message, severity, status, requestId, evidence, recommendation, detection }` |
| 请求详情 | `GET /api/audit/requests/:requestId` | Go 网关 | - | 授权业务或测试的完整记录，含 `kind`、`time`、`alertIds[]` |
| 审计记录 | `GET /api/audit/requests` | Go 网关 | - | 授权请求数组，只读 |
| 风险复核 | `POST /api/risk/reviews` | Go 网关 | `{ alertId, decision, note }`，结论为 `confirmed` 或 `dismissed` | 含 `review` 的更新后告警 |
| 用户列表 | `GET /api/users` | 组员 B 后台 | - | 用户对象数组 |
| 用户详情 | `GET /api/users/:id` | 组员 B 后台 | - | 用户对象 |
| 新增用户 | `POST /api/users` | 组员 B 后台 | `{ username, password, role, dataLevel, department, scope }` | 创建后的用户对象（`201`）；前端密码必填 |
| 更新用户 | `PUT /api/users/:id` | 组员 B 后台 | `{ role?, dataLevel?, department?, scope? }` | 更新后的用户对象 |
| 删除用户 | `DELETE /api/users/:id` | 组员 B 后台 | - | `{ id, deleted }` |
| 角色列表 | `GET /api/roles` | 组员 B 后台 | - | `[{ name, maxAccessLevel, chainRoleOrdinal, permissions, scope }]` |
| 数据分级 | `GET /api/data-levels` | 组员 B 后台 | - | `[{ level, desc, fields, rank }]` |
| 运行配置 | `GET /api/system/settings` | Go 网关 | - | `{ policy:{allowGatewayTests}, runtime:{aiTimeoutMs} }` |
| 保存配置 | `PUT /api/system/settings` | Go 网关 | 同上，至少提供一个已知配置字段 | 更新后的配置 |

`user` 含 `id`、`username`、`role`、`dataLevel`、`department`、`permissions` 与 `scope`，响应不返回密码。除登录和健康检查外，接口需有效 Bearer 令牌。用户/角色/分级/配置仅对 `admin.manage` 开放；态势、告警、审计与复核分别检查职责权限和业务范围。

完整字段说明、校验规则与错误码见 [docs/API接口规范文档.md](../docs/API接口规范文档.md)。本地未实现客户范围校验的 `/api/v1` 全局链上桩接口当前返回 `403`。

检测结果中的 `detection` 包含 `source`、`mode`、`degraded`、`reason`。AI 不可达、超时或响应无效时，网关执行备用规则并返回降级原因；模型未就绪时，AI 服务也会显式返回规则模式与降级状态。页面分别提示降级、能力未知和演示数据，放行结果仍可查看请求详情。

审计页支持按请求编号搜索、查看告警证据、复核记录与建议，并展开授权请求的原始输入及逐层记录。风控测试结果关联 `/risk-review?requestId=...`，包含告警时同时传 `alertId`。历史告警没有原始请求时显示“未关联”。Go 演示记录保存在进程内存中，最多保留 500 个请求；Mock 使用当前浏览器标签页的会话存储，最多保留 200 个请求。关联告警随请求淘汰，详情不存在或已过期时有明确提示，超出授权范围返回 `403`。

### 业务后台接口的校验规则

以下为后端已实现的业务校验，前端可据此做前置提示（但**以服务端校验为准**）：

| 场景 | 结果 |
| --- | --- |
| 新增用户：用户名为空 / 角色非法 / 密级非法 | `400` |
| 新增/更新用户：部门非法、范围超出角色允许部门、柜员取消本人限制 | `400` |
| 新增用户：用户名已存在 | `409` |
| 更新用户：`role`、`dataLevel`、`department` 与 `scope` 均未提供 | `400` |
| 更新/删除用户：用户不存在 | `404` |
| 修改或删除用户：内置 `admin` 账号 | `403` |
| 配置：等待时间不为 500–8000 毫秒整数，或未提供有效字段 | `400` |

**授权边界**：业务部门仅为零售业务部或信贷业务部；柜员／客服限定所属部门的本人业务，风控审核员限定所属部门，审计人员可授权零售/信贷部门子集。系统管理员所属平台运维部，客户业务范围必须为空。各角色可配置 L1–L4，修改密级不会增加角色操作权限。

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
