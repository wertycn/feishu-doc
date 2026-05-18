# feishu-doc

飞书文档与 Markdown 双向转换的 Go SDK 和命令行工具。支持标准 Markdown 语法和 Mermaid 图表的往返无损转换。

## 功能

- **导出**：飞书文档 → Markdown（支持图片下载）
- **导入**：Markdown → 飞书文档（追加 / 全量覆盖）
- **Mermaid**：飞书 AddOns 图表块 ↔ Markdown fenced code block，双向无损
- **知识库**：浏览文档树、创建子文档、复制文档
- **安全**：凭证 AES-256 加密存储，图片下载禁止内网访问

## 安装

### 作为 CLI 工具

```bash
go install github.com/wertycn/feishu-doc/cmd/feishu-doc@latest
```

或从源码编译：

```bash
git clone https://cnb.cool/debug.icu/feishu-doc.git
cd feishu-doc
make build
```

### 作为 Go SDK

```bash
go get github.com/wertycn/feishu-doc
```

```go
import (
    "github.com/wertycn/feishu-doc/core"
    "github.com/wertycn/feishu-doc/core/markdown"
)

// 创建客户端
client := core.NewClient(appID, appSecret, "your-domain.feishu.cn")
client.UserAccessToken = "u-xxx"

// 导出飞书文档为 Markdown
result, err := client.ExportToMarkdown(ctx, docID, &core.ExportOptions{
    WithImages: true,
    OutputDir:  "./output",
})
fmt.Println(result.Markdown)

// Markdown → 飞书文档块
blocks := markdown.Convert([]byte("# Hello\n\nworld"))

// 全量覆盖导入
err = client.OverwriteMarkdown(ctx, docID, mdContent, "doc.md")

// 追加导入
err = client.ImportMarkdown(ctx, docID, mdContent, "doc.md")
```

## 快速开始（CLI）

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

## SDK 包结构

```
github.com/wertycn/feishu-doc/
├── core/                  # 核心 SDK — 飞书文档操作
│   ├── Client             # 飞书 API 客户端
│   ├── ExportToMarkdown   # 飞书文档 → Markdown
│   ├── ImportMarkdown     # Markdown → 飞书文档（追加）
│   ├── OverwriteMarkdown  # Markdown → 飞书文档（全量覆盖）
│   ├── FetchAllBlocks     # 获取文档所有块
│   └── ...                # 创建、复制、知识库操作
├── core/markdown/         # Markdown 解析与转换
│   ├── Convert            # Markdown → 飞书块
│   └── BlocksToMarkdown   # 飞书块 → Markdown（在 core 包中）
├── cmd/feishu-doc/        # CLI 入口
└── cli/                   # CLI 命令实现
```

## 命令参考

### 文档操作

```bash
feishu-doc create --title "新文档"
feishu-doc get --id <doc_id>                               # 纯文本
feishu-doc get --id <doc_id> --blocks                      # 块结构
feishu-doc export --id <doc_id> --file out.md --with-images
feishu-doc import --file doc.md --id <doc_id>              # 追加
feishu-doc import --file doc.md --id <doc_id> --overwrite  # 全量覆盖
feishu-doc import --file doc.md --parent <node_token> --title "标题"
feishu-doc update --id <doc_id> --content "追加内容"
feishu-doc copy --id <node_token> --parent <目标父节点> --name "副本"
```

### 知识库操作

```bash
feishu-doc wiki spaces
feishu-doc wiki info --token <node_token>
feishu-doc wiki tree --space <space_id>
feishu-doc wiki create --parent <node_token> --title "新文档"
```

### 配置与认证

```bash
feishu-doc config set --app-id <id> --app-secret <secret>
feishu-doc config show
feishu-doc auth login
feishu-doc auth status
```

## 支持的文档元素

| 元素 | 导出格式 | 导入 |
| --- | --- | --- |
| 标题 1-6 | `# ~ ######` | ✓ |
| 段落 | 文本 | ✓ |
| 无序/有序列表 | `- item` / `1. item`（支持嵌套） | ✓ |
| 待办事项 | `- [x]` / `- [ ]` | ✓ |
| 代码块 | ` ```lang ``` `（40+ 种语言） | ✓ |
| Mermaid 图表 | ` ```mermaid ``` ` | ✓ |
| 引用 | `> text` | ✓ |
| 高亮块 | `> [!NOTE]` / `[!TIP]` / `[!WARNING]` | ✓ |
| 表格 | Markdown 表格 | ✓ |
| 图片 | `![](url)` | ✓ |
| 分割线 | `---` | ✓ |
| 行内样式 | `**粗体**` `*斜体*` `` `代码` `` `~~删除线~~` | ✓ |
| 链接 | `[text](url)` | ✓ |
| 行内公式 | `$formula$` | ✓ |
| @用户 | `@名称` | ✓ |
| @文档 | `[标题](url)` | ✓ |
| 不支持的块 | `<!-- feishu:type=xxx -->` 注释保留 | — |

## 依赖

| 组件 | 用途 |
| --- | --- |
| [oapi-sdk-go/v3](https://github.com/larksuite/oapi-sdk-go) | 飞书官方 Go SDK |
| [cobra](https://github.com/spf13/cobra) | CLI 框架 |
| [goldmark](https://github.com/yuin/goldmark) | Markdown 解析（GFM 扩展） |

## License

MIT
