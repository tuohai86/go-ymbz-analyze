# Zeabur 快速部署指南

## 🚀 一键部署

本项目已配置为 Zeabur 友好型部署，静态文件（HTML、CSS、JS）会自动嵌入到 Go 二进制文件中。

## 部署步骤

### 1. 推送代码到 Git 仓库

```bash
git add .
git commit -m "准备部署到 Zeabur"
git push origin main
```

### 2. 在 Zeabur 创建项目

1. 访问 [Zeabur 控制台](https://zeabur.com)
2. 点击「New Project」
3. 选择「Connect Git Repository」
4. 选择你的代码仓库

### 3. 配置数据库

#### 选项 A：使用 Zeabur MySQL 服务

1. 在项目中点击「Add Service」
2. 选择「MySQL」
3. 等待 MySQL 启动
4. Zeabur 会自动设置环境变量

#### 选项 B：使用外部数据库

在服务的环境变量中添加：

```env
DB_HOST=your-database-host.com
DB_PORT=3306
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=benz_analysis
SERVER_PORT=8080
```

### 4. 部署 Go 服务

1. Zeabur 会自动检测 Go 项目
2. 自动运行 `go build`
3. 静态文件会自动嵌入二进制
4. 服务自动启动

### 5. 访问应用

- Zeabur 会自动生成一个域名
- 或者绑定你的自定义域名
- 支持自动 HTTPS

## ✅ 部署检查清单

- [ ] 代码已推送到 Git 仓库
- [ ] index.html 和 assets 目录在仓库中（不要加入 .gitignore）
- [ ] 环境变量已配置
- [ ] MySQL 服务已启动
- [ ] 服务状态显示「Running」
- [ ] 可以访问首页（不再返回 404）

## 🔧 常见问题

### Q: 部署后访问首页返回 404

**A:** 这个问题已解决！使用 Go embed 后，静态文件会自动打包到二进制文件中。

### Q: 如何查看日志？

**A:** 在 Zeabur 控制台的「Logs」标签页查看实时日志。

### Q: 如何更新代码？

**A:** 只需推送代码到 Git 仓库，Zeabur 会自动重新构建和部署：

```bash
git add .
git commit -m "更新功能"
git push origin main
```

### Q: 数据库连接失败？

**A:** 检查环境变量是否正确设置，特别是：
- `DB_HOST` - 数据库主机地址
- `DB_PORT` - 端口（默认 3306）
- `DB_USER` - 用户名
- `DB_PASSWORD` - 密码
- `DB_NAME` - 数据库名

### Q: 如何查看当前配置？

**A:** 检查日志，启动时会显示：
```
✅ 配置加载完成
✅ 数据库连接成功
✅ 数据库表迁移完成
🚀 服务器启动在端口: 8080
```

## 🎯 技术实现

本项目使用 **Go 1.16+ embed** 功能：

```go
//go:embed index.html
var indexHTML embed.FS

//go:embed assets
var assetsFS embed.FS
```

这样做的优势：
- ✅ 无需手动复制静态文件
- ✅ 单一二进制文件包含所有资源
- ✅ 部署简单，避免路径问题
- ✅ 完美支持 CI/CD 环境
- ✅ 生产环境零配置

## 📝 环境变量说明

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| DB_HOST | 数据库主机 | localhost | ✅ |
| DB_PORT | 数据库端口 | 3306 | ✅ |
| DB_USER | 数据库用户名 | root | ✅ |
| DB_PASSWORD | 数据库密码 | - | ✅ |
| DB_NAME | 数据库名称 | benz_analysis | ✅ |
| SERVER_PORT | 服务端口 | 8080 | ✅ |

## 🔗 相关文档

- [完整部署指南](./DEPLOY.md)
- [项目说明](./README.md)
- [Zeabur 官方文档](https://zeabur.com/docs)
