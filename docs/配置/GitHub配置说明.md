# GitHub 配置说明

## 概述

系统中与 GitHub 相关的功能主要包括：**数据库备份到 GitHub**（将备份文件上传到 GitHub 仓库）。以下说明如何在后台配置 GitHub 以实现自动或手动备份。

## 一、GitHub 侧准备

### 1. 创建仓库

1. 登录 [GitHub](https://github.com/) → 新建仓库（New repository）。
2. 仓库名自定（如 `cboard-backup`），选择 Private 或 Public，**不要**勾选「Initialize with README」以免与系统首次推送冲突（若已勾选也可，系统会按实际逻辑覆盖或更新文件）。
3. 记下仓库的 **Owner** 和 **Repo** 名称，例如：`your-username/cboard-backup`。

### 2. 创建 Access Token

1. GitHub 右上角头像 → **Settings** → 左侧 **Developer settings** → **Personal access tokens** → **Tokens (classic)**。
2. **Generate new token (classic)**，勾选权限：
   - 至少勾选 **repo**（完整仓库读写），若仅备份到私有库，勾选 repo 即可。
3. 生成后**立即复制** Token，只显示一次；若丢失需重新生成。

## 二、系统后台配置

### 1. 进入备份设置

- **管理员后台** → **系统设置** → **备份设置**（或「备份」相关标签）。
- 在「备份目标」或类似选项中，选择 **GitHub**。

### 2. 填写 GitHub 信息

- **启用 GitHub 备份**：打开开关。
- **Access Token**：粘贴上述 Personal access token（classic）。
- **仓库所有者（Owner）**：GitHub 用户名或组织名，例如 `your-username`。
- **仓库名（Repo）**：仓库名称，例如 `cboard-backup`。
- **分支（如有）**：一般为 `main` 或 `master`，按仓库默认分支填写。
- 若有「备份文件名」「备份目录」等选项，可按需填写（如固定为 `cboard.db` 或带日期的文件名）。

### 3. 保存并测试

- 保存配置后，若有「立即备份」或「测试备份」按钮，可执行一次，然后在 GitHub 仓库中查看是否出现备份文件（如 `cboard.db` 或带时间戳的文件）。
- 若配置了定时备份，系统会按计划将数据库等文件上传到该仓库。

## 三、安全与注意

1. **Token 权限**：仅授予必要权限（如 repo）；不要使用账号密码。
2. **Token 保管**：Token 等同于密码，不要提交到代码或公开文档；仅在后台配置中填写。
3. **私有仓库**：建议使用 Private 仓库存放备份，避免数据泄露。
4. **网络**：服务器需能访问 `api.github.com` 和 `github.com`；若在国内服务器，可能需考虑网络或代理。
5. **频率**：定时备份不宜过于频繁，避免触发 GitHub API 限制；一般每日 1～2 次即可。

## 四、常见问题

- **推送失败：401 Unauthorized**  
  检查 Token 是否正确、是否过期、是否具有 repo 权限。

- **推送失败：404 Not Found**  
  检查 Owner 与 Repo 名称是否正确、仓库是否存在、Token 所属账号是否有该仓库权限。

- **推送失败：连接超时 / 无法访问**  
  检查服务器到 GitHub 的网络（防火墙、DNS、代理）；必要时在服务器上配置 HTTP/HTTPS 代理。

## 相关文档

- [README - 数据库备份](../README_zh.md#-数据库备份)（若项目 README 中有备份章节）
