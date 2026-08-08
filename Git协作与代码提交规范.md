# Git 协作与代码提交规范

> 适用项目：链安智御 —— 面向银行大模型生产应用的全链路安全管控平台（三人比赛项目）  
> 适用对象：全体组员  
> 目标：统一分支、提交、合并流程，避免冲突和误操作，保证代码可回溯、可评审。

---

## 1. 仓库与权限约定

- 仓库默认设为 **Private（私有）**，仅组员可见；比赛结束后如需公开再修改。
- 三名组员均添加为协作者：GitHub 仓库页面 → **Settings → Collaborators → Add people**，输入队友的 GitHub 用户名。
- `main` 是唯一长期分支，开启分支保护（Settings → Branches → Add branch ruleset / rule）：
  - Require a pull request before merging（必须通过 PR 合并）
  - Require approvals = 1（至少 1 名组员评审）
  - 禁止直接 push 到 main

---

## 2. 分支规范

| 分支类型 | 命名规则 | 示例 | 说明 |
| --- | --- | --- | --- |
| 主分支 | `main` | `main` | 稳定可用，只通过 PR 合并 |
| 功能 | `feature/功能名` | `feature/access-control` | 新功能 |
| 修复 | `fix/问题描述` | `fix/redis-timeout` | 缺陷修复 |
| 文档 | `docs/内容` | `docs/git-guide` | 文档更新 |
| 重构 | `refactor/内容` | `refactor/api-structure` | 代码重构 |

规则：

- 分支名使用英文小写，单词之间用 `-` 连接，不要用中文。
- 一个分支只做一件事，任务完成后通过 PR 合并，并删除远程分支。
- 禁止直接在 `main` 上开发，紧急修改也要走 PR。

---

## 3. Commit 提交规范

统一使用 **Conventional Commits（约定式提交）** 格式：

```
type(scope): subject
```

- `type`：提交类型（见下表）
- `scope`（可选）：影响模块，如 `backend`、`chain`、`docs`、`config`
- `subject`：一句话描述，简洁明确

常用 type：

| type | 用途 | 示例 |
| --- | --- | --- |
| `feat` | 新功能 | `feat(chain): 新增合规策略合约` |
| `fix` | 修复缺陷 | `fix(cache): 修复 Redis 连接泄漏` |
| `docs` | 文档 | `docs: 补充接口说明` |
| `style` | 格式调整（不改逻辑） | `style: 统一缩进` |
| `refactor` | 重构 | `refactor(api): 拆分路由` |
| `test` | 测试 | `test(api): 增加鉴权用例` |
| `chore` | 构建、依赖、工具 | `chore: 更新依赖` |

提交要求：

1. **一次提交只做一件事**（原子提交），不要把无关改动混在一起。
2. subject 统一使用中文，动词开头，控制在 50 字以内，说明"做了什么"。
3. 每次提交都应是**可运行、可独立回滚**的状态。
4. 提交前先 `git diff` 检查，确认没有多余或敏感内容。

示例：

```
feat(chain): 新增合规策略合约
fix(backend): 修复鉴权中间件空指针
docs: 更新 Git 协作规范
chore: 添加 .gitignore
```

---

## 4. 日常工作流程

### 4.1 开始新任务

```bash
git checkout main
git pull --rebase origin main       # 先同步最新代码
git checkout -b feature/xxx         # 新建功能分支
```

### 4.2 提交代码

```bash
git status                          # 确认改动内容
git diff                            # 检查具体改动
git add <具体文件>                  # 只添加相关文件，不要用 git add -A
git commit -m "feat(xxx): 描述"
```

### 4.3 推送并合并

```bash
git push -u origin feature/xxx
```

然后在 GitHub 上创建 Pull Request：

- PR 标题与提交信息格式一致，例如 `feat(chain): 新增合规策略合约`
- 描述写清：**改了什么 / 为什么改 / 怎么验证**
- 指派 1 名组员 review，通过后再合并到 `main`
- 合并后删除远程分支（GitHub 会提示）

### 4.4 同步最新代码

```bash
git fetch origin
git rebase origin/main              # 或 git merge origin/main
```

如果产生冲突，先与涉及组员沟通，再解决冲突，不要单方面覆盖对方逻辑。

---

## 5. Pull Request 评审要点

Review 时至少检查：

- [ ] 功能是否按需求实现
- [ ] 是否包含密钥、密码、私钥等敏感信息
- [ ] 是否有无关文件（node_modules、缓存、本地配置）
- [ ] 是否通过编译 / 测试（backend：`go build ./...`；chain：`npx hardhat compile`）
- [ ] 命名、注释、风格是否与团队一致
- [ ] 是否修改了不该动的文件（如别人的模块）

---

## 6. 禁止提交的内容

以下内容一律不得进入仓库（已在 `.gitignore` 中配置）：

- 环境变量与密钥：`.env`、`*.pem`、`*.key`、`*.p12`、证书文件
- 依赖与构建产物：`node_modules/`、`chain/artifacts/`、`chain/cache/`、后端编译产物
- IDE / 系统文件：`.idea/`、`.vscode/`、`.DS_Store`、`Thumbs.db`
- 日志与临时文件：`*.log`、`tmp/`
- 超过 10MB 的大文件（改用网盘或 Git LFS）

> 若误提交了敏感文件：**立即更换密钥/密码**，并从提交历史中清除（可用 `git filter-repo` 或联系仓库负责人处理），不能只删文件后继续提交。

---

## 7. 首次推送（一次性初始化）

如果从本地已有代码开始（推荐在 GitHub 上创建**空仓库**，不勾选任何初始化选项）：

```bash
git init
git add .
git commit -m "chore: 初始化项目"
git branch -M main
git remote add origin https://github.com/<用户名>/<仓库名>.git
git push -u origin main
```

> 注意：如果远程仓库已勾选"Add a README"等初始化选项，首次推送前需要先执行 `git pull --rebase origin main`，否则会因历史不一致而推送失败。

---

## 8. 红线与注意事项

- 不要在 `main` 上直接 commit / push
- 不要随意 `git push -f`（仅限个人分支且确认无人使用）
- 不要用 `git add -A` 或 `git commit -am` 把无关文件一起提交
- 提交前检查 `git status`，避免误传本地配置文件
- 遇到不确定的合并或冲突，先沟通再操作
