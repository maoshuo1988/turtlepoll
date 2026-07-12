# 主站消息 yaml 配置说明

## 1. 配置示例

```yaml
messageNotify:
  enabled: true
  defaultPageSize: 20
  maxPageSize: 100
  renderStrict: true
  readUpdateThrottleSeconds: 0
```

## 2. 字段说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| messageNotify.enabled | bool | true | 主站消息推送总开关 |
| messageNotify.defaultPageSize | int | 20 | 列表默认分页大小 |
| messageNotify.maxPageSize | int | 100 | 列表最大分页大小 |
| messageNotify.renderStrict | bool | true | 渲染后存在未替换占位符时失败 |
| messageNotify.readUpdateThrottleSeconds | int | 0 | 已读更新频控，0 表示不限制 |

## 3. 生效规则

- `enabled=false` 时公共推送方法返回 `MESSAGE_NOTIFY_DISABLED`。
- `enabled=false` 不影响用户查询历史消息。
- `defaultPageSize <= 0` 时使用 20。
- `maxPageSize <= 0` 时使用 100。
- 请求 `limit > maxPageSize` 时按 `maxPageSize` 执行。
- `renderStrict=true` 时，渲染结果存在 `{` 或 `}` 直接失败。

## 4. 与现有配置关系

- 不复用 `aiChat` 配置。
- 不复用旧邮件通知配置。
- 不影响旧 `SysConfigEmailNoticeIntervalSeconds`。

## 5. 环境变量映射

若项目后续支持环境变量覆盖，建议映射：

| 环境变量 | 配置项 |
|----------|--------|
| MESSAGE_NOTIFY_ENABLED | messageNotify.enabled |
| MESSAGE_NOTIFY_DEFAULT_PAGE_SIZE | messageNotify.defaultPageSize |
| MESSAGE_NOTIFY_MAX_PAGE_SIZE | messageNotify.maxPageSize |
| MESSAGE_NOTIFY_RENDER_STRICT | messageNotify.renderStrict |

## 6. 运营开关

MVP 只支持 yaml 总开关。P1 可把 `messageNotify.enabled` 同步到后台系统配置，并支持灰度业务类型开关。
