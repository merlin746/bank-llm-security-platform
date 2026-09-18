# 链安智御（ChainWise Guard）

面向银行大模型生产应用的全链路安全管控平台。工行杯金融安全服务赛道参赛项目。

> 不是给大模型叠加更多"补丁式"内容过滤，而是为大模型在银行核心业务中的每一次调用建立
> **事前硬拦截、事中语义脱敏、事后链上定责**的全链路安全管控闭环。

## 三大核心创新

| 创新 | 内容 | 支撑技术 |
| --- | --- | --- |
| **创新一** 事前规则硬拦截 | 数据密级与合规策略固化于链上，结合缓存预加载，在回答生成**之前**完成毫秒级访问管控 | Solidity 合约 + Redis 预加载 |
| **创新二** 异步四节点对账 | 访问 / RAG / 推理 / 数仓四节点指纹链上比对，主流程与写链解耦，高并发下精准定责 | RabbitMQ + NodeReconciliation |
| **创新三** AI 语义防护 × 区块链确定性执行 | AI 解决非结构化语言的语义安全与脱敏，区块链保证规则确定执行与多方信任 | Qwen2 / NER / 孤立森林 + FISCO BCOS |

## 技术栈

- **链上**：FISCO BCOS v3.x · Solidity ^0.8.0（+ Foundry 测试）
- **后端**：Go 1.21 · Gin · go-redis · RabbitMQ
- **AI**：Python · FastAPI · PyTorch · Qwen2 · Scikit-learn · SpaCy
- **前端**：Vue 3 · Element Plus · ECharts · Pinia · Vite

## 目录结构

```
backend/     Go 安全网关（cmd、internal/api、internal/cache、internal/mq、internal/contract）
chain/       链上合约与 Foundry 测试（contracts/、test/、foundry.toml）
FastAPI/     AI 安全风控微服务（main.py、app/models、scripts）
frontend/    Vue 3 可视化管理平台
docs/        交付文档（开题报告 / API 规范 / 架构设计书）
```

## 交付文档

| 文档 | 内容 |
| --- | --- |
| [项目开题报告](docs/项目开题报告.md) | 背景痛点、研究现状、目标、研究内容与关键技术、创新点、可行性、分工与进度 |
| [API 接口规范文档](docs/API接口规范文档.md) | 统一响应结构、全部 28 条路由的请求/响应契约、管道阶段契约、指纹计算规范 |
| [系统架构设计书](docs/系统架构设计书.md) | 分层架构、核心模块设计、时序流程、数据设计、降级策略、技术决策与测试保障 |

## 快速开始

### 1. 启动 AI 服务（可选，缺失时网关自动降级）

```bash
cd FastAPI
pip install -r requirements.txt
python -m spacy download zh_core_web_sm
uvicorn main:app --host 0.0.0.0 --port 8000
```

### 2. 启动 Go 安全网关

```bash
cd backend
go build -o chainwise-server ./cmd/server
./chainwise-server config.yaml          # 默认监听 :8080
```

> Redis / RabbitMQ 未启动时网关**仍可运行**（自动降级：无缓存、无异步对账），便于本地演示。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev                             # http://localhost:5173
```

前端默认 `VITE_USE_MOCK=true`，可**脱离后端独立演示**三个页面；联调时将
`frontend/.env.development.local` 的 `VITE_USE_MOCK` 设为 `false`。

## 测试

```bash
# 智能合约（Foundry，73 个用例）
cd chain && forge test

# Go 后端（82 个用例）
cd backend && go test ./...

# 前端构建校验
cd frontend && npm run build
```

## 协作规范

所有代码提交请遵守 [Git协作与代码提交规范.md](./Git协作与代码提交规范.md)。

## 项目信息

- 学校：成都信息工程大学　|　指导老师：刘强
- 成员：段利欢（区块链与链下后端）· 梅涵（网关/业务/前端）· 史晨东（AI 算法）
- 源码仓库：<https://github.com/merlin746/bank-llm-security-platform>
