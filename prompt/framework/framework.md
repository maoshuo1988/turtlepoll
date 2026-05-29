# 文件要求
单个文件不要超过400行
如果内容较多可以拆分多个文件记录章节
# 设计文档结构
00-目录
01-流程图
02-总览与边界
03-特性清单总览
04-特性详情介绍
05-配置与数据模型
06-接口
07-ER关系图

可以增加补充说明的文件如06-技术关键要点等

## 特性清单总览
特性清单以表格形式展开如下字段
| 分类 | 特性 | 状态 | 优先级 | 详情 |
## 特性详情介绍
对关键特性 从技术实现上阐述
### 行数限制
如果特性描述超过400行，不要裁剪特性条目，可以新增特性详情的扩展文件补充比如：
04-特性详情介绍-01
05-特性详情介绍-02
## 流程图
流程图参考文件 (./mermaid-flowchart-prompt.md)

## 接口

### Iris MVC 路由防错提示词

当项目使用 Iris MVC controller 方法名自动映射路由时，设计和实现接口前复制以下提示词：

请检查接口 URL 与 Iris MVC 实际路由是否一致，并严格遵守以下规则：

1. 文档 URL 是最终契约，代码必须服务文档 URL。
2. 不要假设方法名里的下划线 `_` 会自动映射成短横线 `-`。
3. 当 URL 包含短横线，例如 `/ability-options`、`/kill-switch`，必须使用 `BeforeActivation` 显式绑定。
4. 当 URL 包含多级路径，例如 `/market/settle`、`/comment_reward/logs`，必须使用 `BeforeActivation` 显式绑定。
5. 显式绑定格式示例：

```go
func (c *PetController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/ability-options", "GetAbility_options")
}
```

6. controller 文件需要导入：

```go
import "github.com/kataras/iris/v12/mvc"
```

7. 新增接口后必须用文档 URL 做本地 curl 验证，不能只跑单测。
8. curl 返回 HTML 首页通常表示路由未命中并被 SPA fallback 接住；返回 JSON 的“请先登录/无权限”才表示已经命中后端。
9. 对 `/api/admin/**` 接口，验证时同时检查认证方式：cookie 名通常是 `bbsgo_token`，也可用 `Authorization: Bearer <token>`。
10. 如果已有接口使用自动映射且 URL 含下划线，可以保留；但新增文档 URL 使用短横线时必须显式绑定，避免线上 404 或 HTML fallback。
