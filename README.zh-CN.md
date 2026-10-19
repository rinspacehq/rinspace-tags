# 芥子环标签系统

[English](README.md) · [rinspace.com](https://rinspace.com)

芥子环标签系统是芥子环组织知识关系的核心。这里的 Tag 不是附在内容上的普通字符串；
它具有稳定身份，可以同时成为分类节点、知识条目、引用目标和受治理关系的端点。

当前已发行内核提供：

- 永不以名称或 slug 代替的正整数稳定 Tag ID；
- Unicode 规范化名称与互不重叠的直接父级语境；
- 支持多个直接父级的有向无环分类图；
- 只负责名称解析、不会变成独立 Tag 的别名；
- 合并后仍保留原身份的生命周期后继关系；
- 具有可重放幂等身份的合并、弃用、归档与恢复确定性迁移计划；
- 从无环 `requires` 开始的受治理知识陈述；
- 确定性的图校验、祖先与后代投影。

芥子环不设置硬编码的 `tag_type`。知识角色来自语境、明确关系和受治理词表。

## 架构与技术栈

本仓库是**可嵌入的 Go 领域库**，不是前端框架，也不是已经完整交付的独立标签服务。
核心不依赖 HTTP 框架、数据库、芥子环账号或渲染服务。

| 层次 | 当前实现 | 归属 |
| --- | --- | --- |
| 名称与关系 | Go 1.24+、`golang.org/x/text`；稳定 ID、名称语境、多父 DAG、别名与关系校验 | 公开 `core/graph` |
| 生命周期计划 | 纯 Go 命令、重放校验与确定性状态迁移计划 | 公开 `core/lifecycle` |
| 本地校验 | Go CLI 与版本化 JSON 契约 | 公开 `cmd/rinspace-tagsctl`、`contracts` |
| 网站与 API | React + TypeScript + Vite；Go + Gin | [Rinspace Web](https://github.com/rinspacehq/rinspace-web) 与芥子环宿主应用，不属于本核心 |
| 存储与集成 | PostgreSQL，使用 Go `database/sql`；权限、事务、审计/outbox 与发布编排 | 芥子环私仓适配层 |
| Wiki 源码与渲染 | [Gitea](https://github.com/rinspacehq/gitea) 的 Git 仓库；[Rinspace Renderer](https://github.com/rinspacehq/rinspace-renderer) | 独立项目，经宿主 Control Plane 接入 |

当前采用进程内 Go module 消费，不是另起一个 Tags 微服务。下图展示现有边界；
分类图存储在 PostgreSQL 中，不要求另部署图数据库。

```mermaid
flowchart TB
    subgraph HOST["芥子环宿主应用"]
        WEB["React / TypeScript / Vite"] --> API["Go / Gin API"]
        API --> ADAPTER["私有适配层：身份、策略与事务"]
        ADAPTER --> DB[(PostgreSQL)]
        ADAPTER --> CP["Control Plane：发布编排"]
    end
    subgraph PUBLIC["rinspace-tags：公开 Go module"]
        GRAPH["core/graph：名称与关系规则"]
        LIFE["core/lifecycle：请求与状态迁移计划"]
        CLI["本地校验 CLI"] --> GRAPH
    end
    ADAPTER -->|"固定版本 Go module"| GRAPH
    ADAPTER -->|"固定版本 Go module"| LIFE
    CP <--> GIT["Gitea：Wiki Git 历史"]
    CP <--> RENDER["Rinspace Renderer：文档结果"]
```

### 迁移状态

芥子环私仓已经固定消费
[`v0.2.0-rc.1`](https://github.com/rinspacehq/rinspace-tags/releases/tag/v0.2.0-rc.1)，
并实际调用公共名称规范化与生命周期计划接口。这是**部分接入**，不是整个标签系统
已经完成开源迁移。完整治理命令、知识发布、PostgreSQL 参考存储，以及全路径消费与
源码权威切换仍未完成。独立 HTTP 服务、OIDC、TypeScript SDK 和服务镜像属于后续计划，
不是当前候选已交付的组件。

## 本地校验

需要 Go 1.24 或更高版本。

```bash
go test ./...
go run ./cmd/rinspace-tagsctl validate \
  -file examples/synthetic-knowledge-base/graph.json
```

校验器读取一个 `rinspace-tags-graph/v1` JSON 快照，拒绝未知字段，将输入限制在
16 MiB，并输出顺序稳定的校验报告。示例只使用合成数据。

## 集成方式

Go module 是共享实现边界。产品在公共核心外围提供存储、身份、权限、Git 和渲染
适配器。芥子环私仓只按精确语义版本和完整性摘要消费通过审查的发行版，不消费
`main`、`latest`、未审查 PR 或可变工作树。

当前候选尚未包含 PostgreSQL 参考存储、独立 HTTP 服务、Gitea/Renderer 适配器、
OIDC 和 TypeScript SDK。这些能力随后接入同一核心，不会维护另一套 Tag 模型。

`core/lifecycle` 校验生命周期意图并生成纯状态迁移计划。消费方适配层负责行锁、
产品名称语境检查、关联变更、审计/outbox 以及外部仓库动作，并把它们放在自己的
事务边界内执行。

## 关系语义

| 关系 | 含义 | 可编辑事实 |
| --- | --- | --- |
| `child -> parent` | 分类语境 | 直接边 |
| alias | 名称解析 | 别名到 Tag 的绑定 |
| successor | 合并或弃用跳转 | 生命周期绑定 |
| `requires` | 学习或逻辑前置 | 受治理陈述 |
| authored reference | 源文档引用知识 | 发布投影 |
| object association | 外部内容关联 Tag | 产品适配层 |

传递分类关系属于派生投影，不能被当作直接事实编辑。

## 许可证与贡献

Rinspace 自有社区版源码采用 [`AGPL-3.0-only`](LICENSE)。贡献者保留著作权，并通过
[贡献条款](CONTRIBUTION-LICENSE.md)与 [DCO 1.1](DCO)，按 Apache License 2.0
许可有意提交的贡献。参见 [CONTRIBUTING.md](CONTRIBUTING.md) 和
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
