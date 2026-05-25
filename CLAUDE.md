# CLAUDE.md — WeChat Obsidian Bot

微信消息自动同步到 Obsidian 的本地 bot 项目。

## 项目结构

```
├── main.go                        # 入口：bot 初始化、扫码、生命周期
├── config.json                    # 用户配置（不提交到 git）
├── config.example.json            # 配置模板
├── extract_article.py             # 公众号文章提取（feedgrab）
├── transcribe_voice.py            # 语音转录（whisper）
├── 启动.bat                       # Windows 启动脚本
└── internal/
    ├── handler/handler.go         # 消息路由 + 分发
    ├── writer/writer.go           # Markdown 生成 + 写入 Obsidian
    ├── article/extract.go         # 调用 Python 脚本提取文章/转录
    ├── media/download.go          # 媒体文件下载
    └── config/config.go           # 配置加载
```

## 构建与运行

```bash
go build -o wechat-obsidian-bot.exe .
./wechat-obsidian-bot.exe
```

## Git 开发规范

### 分支模型

```
master ──────────────────────────── (稳定/发布)
  ├── dev ──────────────────────── (开发主线)
  │   ├── feature/<name>          (新功能)
  │   ├── fix/<name>              (bug 修复)
  │   └── refactor/<name>         (重构)
  └── hotfix/<name>               (紧急修复，从 master 拉)
```

### 分支用途

| 分支 | 用途 | 从哪拉 | 合到哪 |
|------|------|--------|--------|
| `master` | 稳定版本，只接受 PR | - | - |
| `dev` | 开发集成分支 | `master` | `master` |
| `feature/*` | 新功能开发 | `dev` | `dev` (PR) |
| `fix/*` | 非紧急 bug 修复 | `dev` | `dev` (PR) |
| `hotfix/*` | 紧急线上修复 | `master` | `master` + `dev` |
| `refactor/*` | 代码重构 | `dev` | `dev` (PR) |

### 命名规范

小写英文 + 连字符，简洁描述目的：

```
feature/article-image-download
fix/gbk-encoding-crash
hotfix/login-timeout
refactor/handler-split
```

### 禁止操作

- ❌ **直接 push 到 `master`** — 必须通过 PR
- ❌ **直接 push 到 `dev`** — 必须通过 PR
- ❌ **`git push --force` 到 `master` 或 `dev`**
- ❌ **`git commit --amend` 已推送的 commit**
- ❌ **squash merge** — 保留完整 commit 历史

### 操作流程

```
# 新功能
git checkout dev
git pull
git checkout -b feature/xxx
# ... 开发 ...
git add -A
git commit -m "feat: xxx"
git push -u origin feature/xxx
# → 创建 PR 到 dev

# Bug 修复
git checkout dev
git pull
git checkout -b fix/xxx
# ... 修复 ...
git commit -m "fix: xxx"
git push -u origin fix/xxx
# → 创建 PR 到 dev

# 紧急修复
git checkout master
git checkout -b hotfix/xxx
# ... 修复 ...
git commit -m "hotfix: xxx"
git push -u origin hotfix/xxx
# → 创建 PR 到 master
# → 合并后同步回 dev: git checkout dev && git merge master && git push
```

### Commit 格式

```
feat: 新功能描述
fix: 修复问题描述
refactor: 重构内容
docs: 文档更新
chore: 杂项/构建
hotfix: 紧急修复
```
