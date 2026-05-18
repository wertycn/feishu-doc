# feishu-doc

飞书文档与 Markdown 双向转换的命令行工具。支持标准 Markdown 语法和 Mermaid 图表的往返无损转换。

## 功能

- **导出**：飞书文档 → Markdown（支持图片下载）
- **导入**：Markdown → 飞书文档（追加 / 全量覆盖）
- **Mermaid**：飞书 AddOns 图表块 ↔ Markdown fenced code block，双向无损
- **知识库**：浏览文档树、创建子文档、复制文档
- **安全**：凭证 AES-256 加密存储，图片下载禁止内网访问

## 安装

```bash
go install github.com/debugicu/feishu-doc/cli/cmd@latest
```

或从源码编译：

```bash
git clone https://cnb.cool/debug.icu/feishu-doc.git
cd feishu-doc
make build
```

## 快速开始

```bash
# 1. 配置应用凭证
feishu-doc config set --app-id cli_xxx --app-secret xxx

# 2. 用户授权（浏览器登录）
feishu-doc auth login

# 3. 导出文档为 Markdown
feishu-doc export --id <doc_id> --file output.md --with-images

# 4. 编辑后全量覆盖导入
feishu-doc import --file output.md --id <doc_id> --overwrite
```

## 飞书应用配置

1. 打开 [飞书开放平台](https://open.feishu.cn/app) → 创建企业自建应用
2. 记录 App ID 和 App Secret
3. **安全设置** → 重定向 URL 添加：`http://127.0.0.1:9876/callback`
4. **权限管理** → 开通以下权限：

| 权限标识 | 说明 |
| --- | --- |
| `docx:document` | 查看、创建、编辑新版文档 |
| `wiki:wiki` | 查看、创建、编辑知识库 |
| `docs:document.media:upload` | 导入时上传图片 |
| `docs:document.media:download` | 导出时下载图片 |

5. 创建应用版本并发布

## 命令参考

### 文档操作

```bash
# 创建文档
feishu-doc create --title "新文档"

# 获取文档内容
feishu-doc get --id <doc_id>              # 纯文本
feishu-doc get --id <doc_id> --blocks     # 块结构

# 导出为 Markdown
feishu-doc export --id <doc_id> --file out.md
feishu-doc export --id <doc_id> --file out.md --with-images

# 导入 Markdown
feishu-doc import --file doc.md --id <doc_id>              # 追加
feishu-doc import --file doc.md --id <doc_id> --overwrite  # 全量覆盖
feishu-doc import --file doc.md --parent <node_token> --title "标题"  # 知识库新建

# 追加文本
feishu-doc update --id <doc_id> --content "追加内容"

# 复制文档
feishu-doc copy --id <node_token> --parent <目标父节点> --name "副本"
```

### 知识库操作

```bash
feishu-doc wiki spaces                        # 列出知识库
feishu-doc wiki info --token <node_token>     # 节点信息
feishu-doc wiki tree --space <space_id>       # 文档树
feishu-doc wiki create --parent <node_token> --title "新文档"
```

### 配置管理

```bash
feishu-doc config set --app-id <id> --app-secret <secret>
feishu-doc config show
feishu-doc auth login
feishu-doc auth status
```

## 支持的文档元素

### 导出（飞书 → Markdown）

| 元素 | 输出格式 |
| --- | --- |
| 标题 1-6 | `# ~ ######` |
| 正文 | 段落文本 |
| 无序/有序列表 | `- item` / `1. item`（支持嵌套） |
| 待办事项 | `- [x]` / `- [ ]` |
| 代码块 | ` ```lang ``` `（40+ 种语言） |
| Mermaid 图表 | ` ```mermaid ``` ` |
| 引用 | `> text` |
| 高亮块 | `> [!NOTE]` / `[!TIP]` / `[!WARNING]` 等 |
| 表格 | Markdown 表格 |
| 图片 | `![](url)` 或下载到本地 |
| 分割线 | `---` |
| 行内样式 | `**粗体**` `*斜体*` `` `代码` `` `~~删除线~~` `[链接](url)` |
| 行内公式 | `$formula$` |
| @用户 | `@名称` |
| @文档 | `[标题](url)` |
| 不支持的块 | `<!-- feishu:type=xxx -->` 注释保留 |

### 导入（Markdown → 飞书）

支持标准 Markdown + GFM 扩展（表格、任务列表、删除线），Mermaid 代码块自动转为飞书图表。

## 依赖

| 组件 | 用途 |
| --- | --- |
| [oapi-sdk-go/v3](https://github.com/larksuite/oapi-sdk-go) | 飞书官方 Go SDK |
| [cobra](https://github.com/spf13/cobra) | CLI 框架 |
| [goldmark](https://github.com/yuin/goldmark) | Markdown 解析（GFM 扩展） |

## License

MIT
