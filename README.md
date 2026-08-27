# 口述史资料公开授权工作台

本项目为档案整理员、伦理复核员和资料公开负责人提供单进程浏览器工作台，用于把一份口述史资料从建档推进到公开授权。系统强制执行知情同意边界、不可变转写、敏感片段处置、异议闭环、候选冻结和授权验真规则；越权范围、过期或撤回同意、陈旧 revision、未处置敏感项和未关闭异议都会阻断后续操作。

## 状态流程

案卷按以下业务状态推进：

`草稿` → `同意已核验` → `治理中` / `待整改` → `可冻结` → `已冻结` → `已授权`

每次成功写操作都会递增 `revision`。客户端必须提交 `expectedRevision` 和 `idempotencyKey`；陈旧修改会被拒绝，相同幂等键的重试会返回最初结果。成功与失败操作均进入连续摘要审计链，失败操作不会改变案卷聚合。

## 数据与恢复

默认数据目录为 `./data`，可以通过 `-data` 指定其他本地目录。转写、同意、治理、冻结和凭据等证据以内容摘要命名并不可变保存；案卷清单通过临时文件 `Sync` 后原子 `Rename`。启动时会校验 `schemaVersion`、对象摘要、清单引用、冻结摘要和连续审计序号，发现损坏时拒绝启动，未提交的临时文件不会成为可见事实。

## 构建与运行

要求 Go 1.22 或更高版本，不需要 Node 或第三方 Go 依赖。

```sh
go build ./cmd/server
go run ./cmd/server
```

默认只监听 `127.0.0.1:19081`。可显式指定其他回环端口：

```sh
go run ./cmd/server -addr=127.0.0.1:19123 -data=./data
```

也可以设置纯数字 `PORT`，服务会绑定 `127.0.0.1:<PORT>`。显式 `-addr` 的优先级高于 `PORT`。非回环地址和非法端口会被拒绝。浏览器访问服务根路径 `/` 即可使用完整工作台。

## 测试与自检

运行全部测试：

```sh
go test ./...
```

运行有界端到端自检：

```sh
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

自检会创建临时数据目录，启动真实回环 HTTP 监听，通过同源 JSON API 完成建档、同意核验、两版转写、敏感治理、异议整改、冻结、授权、凭据验真、陈旧 revision 阻断及撤回同意后的凭据失效检查，随后主动关闭服务并清理临时数据。

## HTTP API

浏览器工作台使用 `/api/v1` 下的同源 JSON API。写请求必须使用 `Content-Type: application/json`，请求体上限为 1 MiB，并统一携带操作者、`expectedRevision` 与 `idempotencyKey`。主要资源包括案卷、同意证据、转写版本、敏感项、伦理异议、冻结清单和公开授权凭据；`GET /healthz` 提供就绪探测。

案卷详情会返回 `consentCoverage` 同意覆盖矩阵、`transcriptImpact` 转写影响汇总和 `freezePreview` 冻结候选摘要；也可通过 `GET /api/v1/cases/{caseID}/freeze-preview` 单独读取候选。敏感项支持在 `/findings` 或 `/findings/batch` 一次提交 `items` 批次，服务会返回分类统计和治理阻断项。冻结请求必须携带预览得到的 `confirmedManifestDigest`，摘要变化或 `expectedRevision` 陈旧时会拒绝冻结。
