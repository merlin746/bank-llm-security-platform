# 链安智御 · API 接口规范文档

> **面向银行大模型生产应用的全链路安全管控平台**
> 文档版本：v1.0　|　适用代码版本：`main`
> 维护责任：梅涵（网关/业务/前端）· 段利欢（链下后端）· 史晨东（AI 服务）

---

## 1. 总则

### 1.1 基础约定

| 项目 | 约定 |
| --- | --- |
| 协议 | HTTP/1.1 |
| 生产网关地址 | `http://<gateway-host>:8080` |
| 开发态入口 | `http://localhost:5173/api/**`（Vite 代理转发至 `127.0.0.1:8080`） |
| 请求体格式 | `application/json; charset=utf-8` |
| 响应体格式 | `application/json; charset=utf-8` |
| 字符编码 | UTF-8 |
| 认证方式 | 请求头 `Authorization: Bearer <token>`，令牌由账号密码认证后签发并校验会话 |
| 时间格式 | 秒级 Unix 时间戳（链上字段）；`YYYY-MM-DD HH:mm:ss`（展示字段） |

### 1.2 路由分层说明

后端同时提供两套前缀，这是**刻意的设计**而非历史遗留：

| 前缀 | 定位 | 消费方 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/**` | **核心业务接口** | 链下服务、外部系统 | 对接链上合约与 Redis，字段命名为链上结构的镜像（如 `request_id`、`max_access_level`） |
| `/api/**` | **前端联调接口** | Vue 3 前端 | 字段命名为前端消费的驼峰形式（如 `requestId`、`dataLevel`），由网关聚合链上与 AI 能力后返回 |

> 迁移建议：正式上线前应将前端接口统一收敛到 `/api/v1`，并保留 `/api` 作为兼容层。

### 1.3 统一响应结构

所有接口（含错误响应）均返回统一信封：

```jsonc
{
  "code": 0,           // 业务状态码，0 = 成功；非 0 = 业务失败
  "message": "success",// 面向开发者的信息
  "msg": "ok",         // 面向用户的提示（前端优先展示此字段）
  "data": { }          // 业务数据；错误响应时该字段被省略（omitempty）
}
```

**成功响应**

```json
{ "code": 0, "message": "success", "msg": "ok", "data": { "total": 7 } }
```

**错误响应**

```json
{ "code": 404, "message": "user not found", "msg": "user not found" }
```

### 1.4 HTTP 状态码

| 状态码 | 含义 | 使用场景 |
| --- | --- | --- |
| `200` | 成功 | 查询、更新、删除成功 |
| `201` | 已创建 | `POST /api/users` 创建成功 |
| `400` | 请求参数错误 | 缺少必填字段、字段取值非法、JSON 格式错误 |
| `401` | 未认证 | 令牌缺失或失效（前端自动跳转登录页） |
| `403` | 禁止访问 | 操作被策略拒绝（如删除内置 admin 账号） |
| `404` | 资源不存在 | 用户 ID / 请求 ID 不存在 |
| `409` | 资源冲突 | 用户名重复 |
| `500` | 服务内部错误 | 服务端异常 |
| `502` | 上游服务不可用 | AI 微服务（FastAPI）不可达 |

> **注意**：HTTP 状态码表达**传输/资源层面**的结果，`code` 表达**业务层面**的结果，两者独立。前端拦截器以 `code !== 0` 判定业务失败。

### 1.5 路由总览

共 40 条路由：`/api/v1` 12 条，`/api` 28 条。除登录与健康检查外，均要求有效会话及对应操作权限。

角色仅决定操作权限，数据记录还须同时满足账号 `dataLevel`、`scope.departments` 和 `scope.ownerOnly`。系统管理员没有客户业务读取权限，即使其账号为 L4 也不能查看客户记录。

**本地实现边界**：下文保留 `/api/v1` 的链上目标契约。当前全局链上桩接口 `/audit/stats`、`/audit/requests`（含详情）、`/audit/anomalies`、`/permission/users/:address`、`/permission/check-access`、`/policy/active` 与 `/policy/rules` 尚未按客户范围校验，已认证调用当前返回 `403`，不会返回全局客户数据。健康检查正常开放，AI 接口要求 `simulation.run` 并受演示测试开关控制。

---

## 2. 核心业务接口（`/api/v1`）

### 2.1 健康检查

#### `GET /api/v1/health`

检查网关自身及其依赖的可用状态。

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `service` | string | 固定为 `chainwise-backend` |
| `status` | string | 固定为 `ok` |
| `redis` | string | `connected` / `disconnected`；**Redis 客户端未初始化时该字段不出现** |

**示例**

```json
{
  "code": 0, "msg": "ok",
  "data": { "service": "chainwise-backend", "status": "ok", "redis": "connected" }
}
```

---

### 2.2 审计溯源

#### `GET /api/v1/audit/stats`

获取对账异常统计，用于审计溯源页顶部指标。

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `total_records` | uint64 | 已创建的请求记录总数 |
| `total_anomalies` | uint64 | 检测到的异常节点次数 |
| `reconciled_count` | uint64 | 已完成对账的请求数 |

---

#### `GET /api/v1/audit/requests`

分页查询请求列表。

**查询参数**

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `offset` | int | `0` | 起始偏移量 |
| `limit` | int | `20` | 每页条数 |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `items` | object[] | 对账结果列表（元素结构见 `ReconciliationResult`） |
| `total` | int64 | 记录总数 |
| `page` | int | 当前页码（1 起） |
| `size` | int | 每页条数 |

---

#### `GET /api/v1/audit/requests/:requestId`

查询单个请求的对账详情，审计溯源页拓扑图的数据源。

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `requestId` | string | 请求唯一标识 |

**响应 `data`**

```jsonc
{
  "reconciliation": {
    "request_id": "REQ-20260826-0003",
    "consistent": false,               // 四节点指纹是否一致
    "consensus_hash": "0xabcdef",      // 多数派共识指纹
    "anomalous_nodes": [1],            // 离群节点序号（0=访问 1=RAG 2=推理 3=数仓）
    "reconciled_at": 1700000100        // 对账完成时间
  }
}
```

**错误**：请求不存在返回 `404`。

---

#### `GET /api/v1/audit/anomalies`

获取近期异常事件列表（告警日志）。

**查询参数**

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `count` | int | `20` | 返回条数 |

**响应 `data`**：异常项数组

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `request_id` | string | 发生异常的请求 ID |
| `node_type` | string | 节点类型：`ACCESS` / `RAG` / `INFERENCE` / `DATA_WAREHOUSE` / `UNKNOWN` |

> **实现说明**：优先读 Redis 异常列表缓存（LRANGE），缓存未命中时回源链上查询。缓存保留最近 100 条（LPUSH + LTRIM）。

---

### 2.3 权限管理

#### `GET /api/v1/permission/users/:address`

查询指定地址的用户权限。

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `address` | string | 用户链上地址，如 `0xoperator001` |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `address` | string | 用户地址 |
| `role` | int | 角色序号，见 [附录 A](#附录-a-链上枚举定义) |
| `max_access_level` | int | 可访问最高密级序号，见 [附录 A](#附录-a-链上枚举定义) |
| `active` | bool | 账号是否启用 |

**错误**：用户不存在返回 `404`。

> **实现说明**：优先读 Redis 用户权限缓存，未命中回源链上。

---

#### `POST /api/v1/permission/check-access`

校验某用户对某密级数据的访问权限。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_addr` | string | 是 | 用户地址 |
| `data_level` | int | 是 | 目标数据密级序号（0–4） |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `allowed` | bool | 是否放行 |
| `reason` | string | 拒绝原因；放行时为空字符串 |

**示例**

```jsonc
// 请求
{ "user_addr": "0xoperator001", "data_level": 3 }

// 响应
{ "code": 0, "msg": "ok", "data": { "allowed": false, "reason": "data level exceeds user clearance" } }
```

**错误**：缺少必填字段返回 `400`。

---

### 2.4 策略管理

#### `GET /api/v1/policy/active`

获取当前生效的合规策略版本。

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `version_id` | uint64 | 版本号（自增）；`0` 表示尚无生效版本 |
| `description` | string | 版本描述 |
| `rules_root_hash` | string | 规则集合的 Merkle Root Hash |
| `effective_timestamp` | uint64 | 生效时间戳 |
| `enacted` | bool | 是否已生效 |

---

#### `GET /api/v1/policy/rules`

获取当前生效版本中所有启用的规则 ID。

**响应 `data`**：字符串数组

```json
["RULE_ADS_001", "RULE_INVEST_001", "RULE_PRIVACY_001"]
```

---

### 2.5 AI 服务代理

网关将请求体**原样透传**给 FastAPI 微服务（`ai.base_url`，默认 `http://127.0.0.1:8000`），并将上游响应体与 HTTP 状态码一并回传。网关侧超时为 **15 秒**。

> **部署提示**：CPU 推理环境下模型层耗时 3–8 秒，若启用模型层建议将超时提高至 30 秒。

#### `POST /api/v1/ai/prompt/detect`

转发至 `POST {ai_base_url}/api/prompt/detect`，检测 Prompt 注入 / 越狱 / 越权意图。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `text` | string | 是 | 待检测的提示词 |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `is_attack` | bool | 是否判定为攻击 |
| `confidence` | float | 置信度（0–1）；规则层命中时为 `1.0` |
| `reason` | string | 判定依据（规则层会给出命中的正则） |
| `layer` | string | 命中的检测层：`rule` / `model` |
| `detection_mode` | string | 本次实际使用的检测模式：`rules` / `model` |
| `degraded` | bool | 本次语义模型是否不可用（即使规则命中也会返回真实降级状态） |
| `degradation_reason` | string | 依赖缺失、模型未就绪、加载或推理失败的说明；未降级时为空 |

---

#### `POST /api/v1/ai/output/desensitize`

转发至 `POST {ai_base_url}/api/output/desensitize`，执行输出脱敏与合规检测。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `text` | string | 是 | 待脱敏的模型输出 |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `desensitized_text` | string | 脱敏后的文本 |
| `detected_entities` | object[] | 识别到的实体，元素含 `type` / `span` / `value` |
| `compliance` | object | 合规评分：`{ score, violations[], is_compliant }` |

**脱敏规则**

| 实体 | 掩码方式 | 示例 |
| --- | --- | --- |
| 手机号 | 保留前 3 后 4 | `138****5678` |
| 身份证 | 保留前 6 后 4 | `110101********777X` |
| 银行卡 | 保留后 4 | `****1234` |
| NER 实体 | 全掩码 | `***` |

---

#### `POST /api/v1/ai/risk/score`

转发至 `POST {ai_base_url}/api/risk/score`，计算用户行为风险评分。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_id` | string | 是 | 用户标识 |
| `history` | object[] | 是 | 行为历史，元素含 `timestamp` / `type` / `is_attack` / `ip` |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | string | 回显用户标识 |
| `score` | float | 风险分（0–100） |
| `is_anomaly` | bool | 是否判定为异常 |
| `level` | string | 风险等级：`low`（<30）/ `medium`（30–59）/ `high`（≥60） |
| `features` | object | 7 维特征明细 |

**特征维度**

| 特征 | 含义 |
| --- | --- |
| `call_count_1h` | 1 小时内调用次数 |
| `call_count_24h` | 24 小时内调用次数 |
| `night_rate` | 夜间（22:00–06:00）调用占比 |
| `avg_interval` | 平均调用间隔（秒，上限 3600） |
| `unique_ip_count` | 不同 IP 数量 |
| `attack_rate` | 触发安全检测的比例 |
| `request_type_entropy` | 请求类型熵 |

---

## 3. 前端联调接口（`/api`）

### 3.1 认证

#### `POST /api/auth/login`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `username` | string | 是 | 用户名，去除首尾空白后查找用户存储 |
| `password` | string | 是 | 与存储的密码精确匹配，保留首尾空白 |

**响应 `data`**

```jsonc
{
  "token": "<随机会话令牌>",
  "user": {
    "id": 1, "username": "admin", "role": "系统管理员", "dataLevel": "L4",
    "department": "平台运维部", "permissions": ["admin.manage"],
    "scope": { "departments": [], "ownerOnly": false }
  }
}
```

内置演示账号如下，密码均与英文账号同名。账号或密码缺失返回 `400`，未知账号或密码错误统一返回 `401` / `账号或密码错误`。

| 账号 | 角色 | 默认密级 | 默认业务范围 |
| --- | --- | --- | --- |
| `teller` | 柜员／客服 | L1 | 所属零售业务部，且仅限本人业务 |
| `reviewer` | 风控审核员 | L3 | 所属零售业务部 |
| `auditor` | 审计人员 | L2 | 零售与信贷业务部，只读审计 |
| `admin` | 系统管理员 | L4 | 空业务范围，仅管理配置 |

这些是账号默认授权，四个角色都可独立配置 L1–L4。职责、密级、部门范围与本人限制分别计算；客户端提交 `role` 或 `scope` 不会改变会话身份。用户对象与后续 `/auth/me` 都返回 `id`、`username`、`role`、`dataLevel`、`department`、`permissions`、`scope`，不返回密码。

Go 后端签发随机令牌，会话保存在进程内存，8 小时后过期，进程重启后失效；每次请求重新读取账号以执行当前角色、密级及范围校验。Mock 令牌及账号保存在标签页的会话存储；权限修改会撤销该账号的旧会话，需重新登录。两种模式均拒绝不存在或已注销的令牌，Go 另校验会话到期时间。

#### `GET /api/auth/me`

要求有效 Bearer 令牌，返回 `{ "user": <当前用户对象> }`；无效会话返回 `401`。前端导航时以此接口恢复身份。

#### `POST /api/auth/logout`

撤销请求头中的当前会话令牌。后端返回 `{ "loggedOut": true }`；Mock 返回成功信封，前端同时清理本地身份。已撤销令牌不能再次使用。

---

### 3.2 攻防测试

#### `POST /api/gateway/attack-test`

Prompt 注入 / 越狱攻击模拟。网关会**真实调用** AI 输入检测接口；AI 服务不可用时降级为规则层判定，保证演示不中断。

要求 `simulation.run`、有效业务范围及启用的 `policy.allowGatewayTests`，当前仅风控审核员具备此操作；系统管理员不自动获得测试权限。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `prompt` | string | 是 | 待测试的提示词 |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `requestId` | string | 本次测试的请求 ID |
| `kind` | string | 请求类型：`attack` / `access` |
| `time` | string | 请求发生时间，`YYYY-MM-DD HH:mm:ss`，北京时间 |
| `alertIds` | int[] | 关联告警编号；未产生告警时为空数组 |
| `detection` | object | 检测来源、模式、降级状态与原因，见下表 |
| `prompt` | string | 回显提示词 |
| `verdict` | string | 最终裁决：`pass` / `block` |
| `totalLatencyMs` | int | 各阶段耗时之和 |
| `stages` | object[] | 拦截管道各阶段结果，见 [3.4](#34-拦截管道阶段契约核心) |

**`detection`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `source` | string | `ai-service` / `gateway-rules` / `mock` |
| `mode` | string | `rules` / `model` / `mock` / `unknown` |
| `degraded` | bool | 语义检测能力是否降低 |
| `reason` | string | 降级或检测能力说明 |

AI 调用超过 `runtime.aiTimeoutMs`（默认 4000 毫秒）、不可达、返回非 2xx 或无效结果时，网关执行中英文备用规则，返回 `gateway-rules` / `rules` / `degraded: true`。AI 已降级时保留其降级原因。规则放行只代表未命中规则，不代表语义模型完成检测。

---

#### `POST /api/gateway/access-test`

越权访问模拟。

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `role` | string | 否 | 只能与当前登录角色一致；其他角色返回 `403`，省略时取会话角色 |
| `dataLevel` | string | 是 | 请求访问的密级：`L1`–`L4` |
| `action` | string | 否 | 操作描述，如"查询 / 导出" |

**密级判定**：使用已认证账号的实际 `dataLevel`，不通过角色名称推导授权密级。客户端请求密级超过账号密级时返回拦截裁决；测试不读取目标密级的客户数据。

**响应 `data`**：同 `attack-test`，另含 `role`、`action`、`requestedLevel`（测试目标密级）和 `dataLevel`（记录自身密级）。记录还含已认证账号的 `owner` 与 `department`，便于在授权范围内关联查看。

---

### 3.3 安全态势

以下接口要求 `dashboard.read`，所有统计只包含当前账号密级与业务范围内的请求和告警。

#### `GET /api/stats/overview`

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `totalRequests` | int | 今日总请求数 |
| `blockedToday` | int | 今日拦截量 |
| `blockRate` | float | 拦截率（%） |
| `highRiskUsers` | int | 高风险用户数 |

---

#### `GET /api/stats/trend`

**响应 `data`**：时序点数组

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `time` | string | 时间刻度，如 `08:00` |
| `blocked` | int | 该时段拦截量 |
| `passed` | int | 该时段放行量 |

---

#### `GET /api/stats/risk-distribution`

**响应 `data`**：风险类型分布数组

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `type` | string | 风险类型，如 `Prompt 注入` |
| `value` | int | 数量 |

---

#### `GET /api/stats/high-risk-users`

**响应 `data`**：高风险用户数组

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 用户标识 |
| `role` | string | 角色 |
| `riskScore` | int | 风险评分（0–100） |
| `lastAction` | string | 最近行为描述 |
| `level` | string | 风险等级：`高` / `中` |

---

### 3.4 拦截管道阶段契约（核心）

`attack-test` 与 `access-test` 返回的 `stages[]` 是前端可视化拦截管道的核心契约。**阶段数量与顺序固定**：

| # | `key` | `name` | `owner` | 提供方 |
| --- | --- | --- | --- | --- |
| 1 | `auth` | 身份认证 | `B` | 网关 |
| 2 | `input-risk` | AI 输入风险检测 | `C` | AI 服务 |
| 3 | `access-control` | 权限/密级校验 | `A` | 链上策略 + Redis |
| 4 | `infer` | LLM 推理节点 | `-` | 推理节点 |
| 5 | `output-sanitize` | 输出脱敏与合规 | `C` | AI 服务 |

**阶段对象结构**

```jsonc
{
  "key": "input-risk",
  "name": "AI 输入风险检测",
  "owner": "C",              // B=网关 A=链上策略 C=AI服务 -=推理节点
  "status": "block",         // pass | block | skip
  "latencyMs": 12,
  "message": "检测到越狱指令，意图越界",
  "extra": { "riskScore": 92, "riskType": "jailbreak" }
}
```

**`status` 语义**

| 取值 | 含义 | 前端表现 |
| --- | --- | --- |
| `pass` | 该阶段通过 | 绿色左边框 + 对勾图标 |
| `block` | 该阶段拦截，管道短路 | 红色左边框 + 红底 + 叉号图标 |
| `skip` | 因上游已拦截而未执行 | 灰色 + 半透明；`latencyMs` 应为 `0` |

**约束（前端依赖，服务端必须遵守）**

1. `stages` 必须包含且仅包含上述 5 个阶段，顺序不可变；
2. 任一阶段 `status` 为 `block` 时，其后所有阶段必须为 `skip` 且 `latencyMs` 为 `0`；
3. `totalLatencyMs` 必须等于各阶段 `latencyMs` 之和；
4. `verdict` 为 `block` 当且仅当存在至少一个 `block` 阶段。

---

### 3.5 审计溯源

#### `GET /api/audit/topology`

四节点指纹连贯性拓扑，审计溯源页主图数据源。

要求 `audit.read`，节点与对账结果只依据当前授权业务记录生成。审计页面仅提供刷新、查看与证据复制，不执行风险复核或配置修改。

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `nodes` | object[] | 节点数组，见下 |
| `edges` | object[] | 边数组，元素为 `{ from, to }`，端点必须是已存在的节点 `id` |
| `chainStatus` | string | `consistent` / `inconsistent` |
| `alert` | string | 告警文案；一致时可为空 |

**节点对象**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 节点标识：`access` / `rag` / `inference` / `warehouse` |
| `name` | string | 展示名，如 `RAG 检索节点` |
| `hash` | string | 该节点计算的处理指纹 |
| `status` | string | `normal` / `tampered` |
| `x` / `y` | int | 拓扑布局坐标 |

> **前端表现**：`status: "tampered"` 的节点会被渲染为菱形 + 红色粗边框 + 光晕。

---

#### `GET /api/audit/alerts`

**响应 `data`**：告警数组

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | int | 告警序号 |
| `time` | string | 时间，`YYYY-MM-DD HH:mm:ss` |
| `node` | string | 节点展示名 |
| `type` | string | `hash-mismatch` / `privilege` / `jailbreak` |
| `message` | string | 告警内容 |
| `severity` | string | 严重程度：`high` 等 |
| `status` | string | 处置状态：`open` / `blocked` 等 |
| `requestId` | string | 关联请求编号；历史演示记录未保存原始请求时为空 |
| `evidence` | string | 检测依据与命中信息 |
| `recommendation` | string | 处置建议 |
| `detection` | object | 告警对应请求的检测来源及降级状态 |
| `owner` / `department` / `dataLevel` | string | 记录所有者、业务组织与数据密级 |
| `review` | object | 已保存时包含 `{ decision, note, reviewer, time }` |

告警列表与详情要求 `alerts.read`，并按密级、业务部门和本人范围过滤。

**`type` 与前端标签映射**

| `type` | 中文标签 | 标签色 |
| --- | --- | --- |
| `hash-mismatch` | Hash 不一致 | danger |
| `privilege` | 越权 | warning |
| `jailbreak` | 注入攻击 | warning |

---

#### `GET /api/audit/alerts/:id`

返回单条授权告警，字段与告警列表对象一致。不存在或已过期返回 `404`，超出范围返回 `403`。

#### `GET /api/audit/requests`

返回当前授权请求数组，要求 `audit.read` 或 `alerts.read`；记录包含 `requestId`、`owner`、`department`、`dataLevel`、`kind`、`time`、处理结果与关联告警。

#### `GET /api/audit/requests/:requestId`

返回授权业务或攻防请求的完整结果，包含 `requestId`、`owner`、`department`、`dataLevel`、`kind`、`time`、原始输入（`prompt` 或 `role` / `requestedLevel` / `action`）、`verdict`、`totalLatencyMs`、`stages`、`detection` 与 `alertIds`。要求 `audit.read`、`alerts.read` 或 `simulation.run`，并校验记录范围；不存在或已过期返回 `404`，超出授权范围返回 `403`。

前端联调请求与告警在 Go 进程内存中关联保存，最多保留 500 个请求，关联告警随请求淘汰，进程重启后清空。此接口与 `/api/v1/audit/requests/:requestId` 的链上对账详情分别服务于网关测试记录与链上审计。

### 3.6 业务后台 — 用户管理

本节全部接口要求 `admin.manage`，不会授予客户数据查看权限。

#### `GET /api/users`

**响应 `data`**：用户数组

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | int | 用户 ID（自增） |
| `username` | string | 用户名 |
| `role` | string | 角色中文名 |
| `dataLevel` | string | 密级 `L1`–`L4` |
| `department` | string | 所属业务部门；系统管理员为平台运维部 |
| `permissions` | string[] | 该角色允许的操作，如 `business.read`、`risk.review`、`admin.manage` |
| `scope` | object | `{ departments: string[], ownerOnly: bool }`，限定记录范围 |

> **安全约束**：响应中**永不包含**密码字段。

---

#### `GET /api/users/:id`

**路径参数**：`id`（int，必须为正整数）

**响应 `data`**：单个用户对象

**错误**：`id` 非法返回 `400`；用户不存在返回 `404`。

---

#### `POST /api/users`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `username` | string | 是 | 用户名，去除首尾空白后不得为空 |
| `role` | string | 是 | 角色中文名，取值见 [3.7](#37-业务后台--角色与数据分级) |
| `dataLevel` | string | 是 | 密级 `L1`–`L4` |
| `password` | string | 前端必填 | 用于后续登录，不参与任何响应；后端兼容省略，创建后的空密码账号不能登录 |
| `department` | string | 否 | 零售业务部或信贷业务部；系统管理员为平台运维部；省略时使用角色默认部门 |
| `scope` | object | 否 | 业务范围；省略时使用角色与部门的默认范围，可提供合法子集以收窄 |

**业务校验**

| 校验 | 失败返回 |
| --- | --- |
| 用户名为空 | `400` |
| 角色非法 | `400` |
| 密级非法 | `400` |
| 部门非法或范围不符合角色职责 | `400` |
| 柜员／客服取消 `ownerOnly` 本人限制 | `400` |
| 用户名已存在 | `409` |

**成功响应**：`201` + 创建后的用户对象。

**示例**

```jsonc
// 请求
{ "username": "jane", "role": "风控审核员", "dataLevel": "L3", "password": "secret", "department": "零售业务部", "scope": { "departments": ["零售业务部"], "ownerOnly": false } }

// 响应 201
{ "code": 0, "msg": "ok", "data": { "id": 5, "username": "jane", "role": "风控审核员", "dataLevel": "L3", "department": "零售业务部", "permissions": ["business.read", "dashboard.read", "alerts.read", "risk.review", "simulation.run"], "scope": { "departments": ["零售业务部"], "ownerOnly": false } } }
```

---

#### `PUT /api/users/:id`

**请求体**（至少提供一个可更新字段，账号名称不可修改）

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `role` | string | 否 | 新角色 |
| `dataLevel` | string | 否 | 新密级 |
| `department` | string | 否 | 新业务部门 |
| `scope` | object | 否 | 新业务范围；更新角色或部门时，未传范围则使用新职责默认范围 |

**业务校验**

| 校验 | 失败返回 |
| --- | --- |
| 所有可更新字段均未提供 | `400` |
| 角色或密级非法 | `400` |
| 更新后的部门或业务范围不符合职责 | `400` |
| 用户不存在 | `404` |
| 修改内置 `admin` | `403` |

四个角色均允许独立配置 L1–L4；提高账号密级不会增加操作权限或授权部门。柜员／客服范围只能为所属部门且 `ownerOnly: true`；风控审核员只能授权所属部门；审计人员可选择零售/信贷业务部子集；系统管理员只能为空业务范围。

---

#### `DELETE /api/users/:id`

**响应 `data`**

```json
{ "id": 3, "deleted": true }
```

**业务约束**

| 场景 | 返回 |
| --- | --- |
| 删除内置 `admin` 账号 | `403` |
| 用户不存在 | `404` |

> **设计理由**：`admin` 是演示环境的兜底账号，允许删除会导致后台无法登录。接入链上后应改为 `deactivateUser`（停用而非删除），以保留审计痕迹。

---

### 3.7 业务后台 — 角色与数据分级

#### `GET /api/roles`

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 角色中文名 |
| `maxAccessLevel` | string | 可配置的密级边界，四角色均为 L4；实际数据访问以账号密级与范围为准 |
| `chainRoleOrdinal` | int | 对应链上 `Role` 枚举序号 |
| `permissions` | string[] | 角色操作权限 |
| `scope` | object | 该角色默认业务范围，与账号密级独立 |

**当前取值**

| `name` | `maxAccessLevel` | `chainRoleOrdinal` |
| --- | --- | --- |
| 系统管理员 | L4 | 4（ADMIN） |
| 风控审核员 | L4 | 3（MANAGER） |
| 柜员／客服 | L4 | 2（OPERATOR） |
| 审计人员 | L4 | 1（AUDITOR） |

角色与分级接口均要求 `admin.manage`；角色定义以只读方式展示，账号管理可选择对应角色。

| 角色 | `permissions` |
| --- | --- |
| 柜员／客服 | `business.read`、`business.submit` |
| 风控审核员 | `business.read`、`dashboard.read`、`alerts.read`、`risk.review`、`simulation.run` |
| 审计人员 | `business.read`、`alerts.read`、`audit.read` |
| 系统管理员 | `admin.manage` |

---

#### `GET /api/data-levels`

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `level` | string | 密级标识 `L1`–`L4` |
| `desc` | string | 密级名称 |
| `fields` | string | 该密级涵盖的数据字段 |
| `rank` | int | 密级序号（1–4），用于越权判断 |

**当前取值**

| `level` | `desc` | `fields` | `rank` |
| --- | --- | --- | --- |
| L1 | 公开数据 | 无 | 1 |
| L2 | 内部数据 | 基础客户信息 | 2 |
| L3 | 敏感数据 | 账户余额、交易明细 | 3 |
| L4 | 高度敏感 | 身份证、卡号、征信 | 4 |

---

### 3.8 授权业务工作台

#### `GET /api/business/requests`

要求 `business.read`，返回当前授权的业务请求数组。柜员／客服仅返回本人记录；风控和审计账号按密级及业务部门范围过滤。`GET /api/business/requests/:requestId` 返回单个授权业务记录，`GET /api/workspace/requests` 为授权记录兼容入口。

#### `POST /api/business/requests`

要求 `business.submit`，当前仅柜员／客服可提交。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `prompt` | string | 是 | 非空业务内容 |
| `dataLevel` | string | 否 | 业务密级 L1–L4，省略为 L1，不能超过账号授权 |
| `businessType` | string | 否 | 业务类型，如产品与服务咨询 |
| `action` | string | 否 | 操作描述 |

`owner`、角色与业务组织由已认证账号确定，客户端不能伪造。响应为请求对象，含 `result`、`riskTip`、`status`、`verdict`、`requestId`、`alertIds` 及检测记录。未发现风险返回处理结果，命中风险返回拦截结果及提示；记录始终按密级、组织及本人限制授权。

### 3.9 风险复核

#### `POST /api/risk/reviews`

要求 `risk.review`，仅风控审核员可在授权告警范围内操作；审计人员保持只读。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `alertId` | int | 是 | 已存在的授权告警编号 |
| `decision` | string | 是 | `confirmed`（确认风险）或 `dismissed`（排除风险） |
| `note` | string | 是 | 非空复核依据，不超过 1000 字 |

返回更新后的告警，`review` 包含 `{ decision, note, reviewer, time }`，`status` 更新为该复核结论。复核仅保存风险判断，不解除拦截、不提升权限。非法参数返回 `400`，越权返回 `403`，不存在返回 `404`，已复核返回 `409`。

### 3.10 策略与运行配置

#### `GET /api/system/settings` / `PUT /api/system/settings`

两者均要求 `admin.manage`。GET 返回当前配置；PUT 至少提供一个已知字段，并返回保存后的配置：

```json
{
  "policy": { "allowGatewayTests": true },
  "runtime": { "aiTimeoutMs": 4000 }
}
```

`allowGatewayTests` 为布尔值；关闭后 `/api/gateway/**` 与 `/api/v1/ai/**` 的新测试请求返回 `403`。`aiTimeoutMs` 为 500–8000 毫秒整数，控制后续 AI 检测等待时间；超时执行备用规则并显式标注降级。空请求、非法超时值或非布尔开关返回 `400`。当前演示配置保存在 Go 进程内存或 Mock 标签页会话中。

---

## 4. 附录

### 附录 A：链上枚举定义

**Role（角色）**

| 序号 | 名称 | 含义 |
| --- | --- | --- |
| 0 | `NONE` | 无权限 |
| 1 | `AUDITOR` | 审计员 |
| 2 | `OPERATOR` | 普通操作员 |
| 3 | `MANAGER` | 经理 |
| 4 | `ADMIN` | 系统管理员 |

**DataLevel（数据密级）**

| 序号 | 名称 | 含义 |
| --- | --- | --- |
| 0 | `PUBLIC` | 公开 |
| 1 | `INTERNAL` | 内部 |
| 2 | `CONFIDENTIAL` | 机密 |
| 3 | `SECRET` | 秘密 |
| 4 | `TOP_SECRET` | 绝密 |

**NodeType（节点类型）**

| 序号 | 名称 | 含义 |
| --- | --- | --- |
| 0 | `ACCESS` | 访问网关节点 |
| 1 | `RAG` | RAG 检索节点 |
| 2 | `INFERENCE` | 推理节点 |
| 3 | `DATA_WAREHOUSE` | 数仓节点 |

**ReviewStatus（高风险操作审核状态）**

| 序号 | 名称 | 含义 |
| --- | --- | --- |
| 0 | `NONE` | 未提交 |
| 1 | `PENDING` | 待审核 |
| 2 | `APPROVED` | 已通过 |
| 3 | `REJECTED` | 已驳回 |

### 附录 B：节点指纹（Node Hash）计算规范

四节点对账的指纹算法必须链上链下严格一致，定义如下：

```
nodeHash = SHA-256(
    "chainwise.node-hash.v1"   // 域分隔前缀，固定
    || 0x00                    // 字段分隔符
    || requestID               // UTF-8 字节
    || 0x00
    || uint32(nodeType)        // 4 字节大端序
    || 0x00
    || payload                 // UTF-8 字节
)
```

**输出格式**：`0x` + 64 位小写十六进制字符（共 66 字符）。

**设计要点**

1. **域分隔前缀**保证不同用途的指纹不会在相同输入下碰撞；
2. 使用不可打印的 `0x00` 作为字段分隔符（而非 `"|"`），避免字段内容本身包含分隔符导致歧义——例如 `("a|b", "c")` 与 `("a", "b|c")` 用可打印分隔符会得到相同输入串；
3. `nodeType` 以定长 4 字节大端整数参与编码，避免十进制字符串表示带来的歧义（如 `1` 与 `01`）；
4. 任意字段变化（包括节点类型不同）都必须产生不同指纹，这是对账能够定位异常节点的前提。

**对应实现**：`backend/internal/contract/node_reconciliation.go` 中的 `CalculateNodeHash`。

### 附录 C：错误码约定

除 HTTP 状态码外，`code` 字段与 HTTP 状态码保持一致的数值语义：

| `code` | 含义 |
| --- | --- |
| `0` | 成功 |
| `400` | 请求参数错误 |
| `401` | 会话缺失、失效或已撤销 |
| `403` | 操作被拒绝 |
| `404` | 资源不存在 |
| `409` | 资源冲突 |
| `500` | 服务内部错误 |
| `502` | 上游 AI 服务不可用 |

### 附录 D：契约变更流程

由于前端与后端多人并行开发，接口契约变更须遵循以下流程：

1. **先改文档**：更新本文档对应条目，并说明变更原因与影响范围；
2. **补测试**：在 `backend/internal/api/router_contract_test.go` 的 `frontendContractRoutes` 表中同步增删路由；
3. **再改代码**：实现变更后运行 `go test ./internal/api/...`；
4. **通知消费方**：涉及字段增删改时须通知前端/链下消费方。

> `TestRouteInventoryMatchesSpec` 会断言路由总数（当前 40 条）。新增或删除接口时该测试会失败，属预期行为——请同步更新 `expectedRouteCount` 与本文档。

---

*v1.0 · 本规范与 `backend/internal/api` 实现及 `frontend/src/api` 消费代码保持同步。*
