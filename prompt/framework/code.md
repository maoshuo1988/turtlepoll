# 编码规范

# 中间件使用

## GORM 布尔默认值防错提示词

当你在 Go + GORM 中设计创建接口，且模型包含 `bool` 字段并带有 `default` 标签（例如 `gorm:"default:true"`）时，必须先执行以下检查，避免出现“请求传 false，数据库仍写成 true”的问题。

### 必查清单

1. 检查模型字段是否为 `bool` + `default`。
2. 检查创建路径是否直接 `db.Create(&model)`。
3. 若字段业务上允许显式写 `false`，禁止仅依赖普通 `bool` 零值写入。

### 推荐实现口径

1. 优先把该字段改为指针类型：`*bool`。
2. 在创建时显式赋值：
	- `v := form.IsPublic`
	- `ModelField: &v`
3. 提供统一访问方法避免空指针，例如：
	- `IsPublicValue() bool`
	- `nil` 时按业务默认值返回（例如 `true`）。
4. 业务层和控制器层统一使用 `IsPublicValue()`，不要直接解引用。

### 审查提示词（可直接复制）

请检查本次改动是否存在 GORM 布尔默认值陷阱，并严格按下列规则修复：

1. 如果模型字段是 `bool` 且有 `gorm:"default:..."`，必须评估 `false` 是否会被数据库默认值覆盖。
2. 对需要可靠写入 `false` 的字段，优先改为 `*bool` 并在创建时显式赋值指针。
3. 为指针布尔字段补充统一读取方法（如 `XxxValue()`），业务代码统一通过该方法判断。
4. 不允许为了修复单字段问题直接使用 `Select("*")` 全字段写入，除非已明确评估过度写入风险。
5. 修复后必须提供回归验证：
	- 创建接口传 `false`；
	- 数据库落库值为 `false`；
	- 相关列表/详情接口返回与数据库一致。

### 回归命令模板

```bash
# 1) 创建（显式传 false）
curl -sS -X POST "${BASE_URL}/api/xxx/create" \
  -H "Content-Type: application/json" \
  --data-raw '{"isPublic":false}'

# 2) 编译/测试校验
go test ./internal/repositories ./internal/services ./internal/controllers/api
```
