# 快速启动指南 - 使用新迁移的数据库

## 🚀 立即开始

### 1. 停止当前服务（如果正在运行）
```bash
# 查找并停止Go服务
pkill -f "go run cmd/server/main.go"
# 或者
lsof -ti:8000 | xargs kill -9
```

### 2. 启动服务使用新数据库
```bash
# 进入项目目录
cd /Users/apple/Downloads/goweb

# 启动服务（会自动使用 cboard (4).db）
go run cmd/server/main.go
```

### 3. 访问新功能

#### 用户端
- 知识库: http://localhost:5173/knowledge
- Dashboard签到: http://localhost:5173/dashboard

#### 管理端
- 知识库管理: http://localhost:5173/admin/knowledge
- 用户分析: http://localhost:5173/admin/analytics
- 营销活动: http://localhost:5173/admin/promotions

---

## ✅ 已完成的迁移

### 数据库更新
- ✅ 4个新表已创建
- ✅ users表新增2个字段
- ✅ 5个分类已导入
- ✅ 14篇文章已导入
- ✅ 教程已更新为Clash系列

### 备份信息
**备份文件**: `/Users/apple/Downloads/goweb/cboard (4).db.backup.20260302_003604`

如需回滚:
```bash
cp "/Users/apple/Downloads/goweb/cboard (4).db.backup.20260302_003604" "/Users/apple/Downloads/goweb/cboard (4).db"
```

---

## 📱 测试清单

### 基础功能测试
- [ ] 服务启动无错误
- [ ] 用户登录正常
- [ ] Dashboard显示正常

### 新功能测试
- [ ] 知识库列表显示（应该有5个分类）
- [ ] 文章详情可以打开
- [ ] 签到按钮可以点击
- [ ] 签到成功后余额增加

### 管理端测试
- [ ] 知识库管理页面打开
- [ ] 可以创建/编辑文章
- [ ] 用户分析页面显示数据
- [ ] 营销活动页面可以创建活动

### 移动端测试
- [ ] 手机浏览器访问正常
- [ ] 抽屉弹窗全屏显示
- [ ] 按钮大小合适
- [ ] 输入框不会自动缩放

---

## 🔍 验证命令

### 检查表是否创建
```bash
sqlite3 "/Users/apple/Downloads/goweb/cboard (4).db" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
```

### 检查知识库数据
```bash
# 查看分类
sqlite3 "/Users/apple/Downloads/goweb/cboard (4).db" "SELECT * FROM knowledge_categories;"

# 查看文章数量
sqlite3 "/Users/apple/Downloads/goweb/cboard (4).db" "SELECT COUNT(*) FROM knowledge_articles;"
```

### 检查users表字段
```bash
sqlite3 "/Users/apple/Downloads/goweb/cboard (4).db" "PRAGMA table_info(users);" | grep telegram
```

---

## ⚠️ 常见问题

### Q: 服务启动报错"table not found"
**A**: 检查数据库文件路径是否正确，确保使用的是 `cboard (4).db`

### Q: 知识库页面显示空白
**A**: 检查浏览器控制台是否有API错误，确认后端服务正常运行

### Q: 签到按钮点击无反应
**A**:
1. 检查是否已登录
2. 检查今天是否已经签到过
3. 查看浏览器控制台错误信息

### Q: 移动端显示不正常
**A**:
1. 清除浏览器缓存
2. 确保前端已重新构建
3. 检查是否导入了移动端优化CSS

---

## 📊 新功能说明

### 1. 签到系统
- 每日签到获得0.1-1元随机奖励
- 奖励自动添加到账户余额
- 记录在余额日志中
- 支持连续签到统计

### 2. 知识库系统
- 5个分类：新手入门、客户端教程、常见问题、进阶使用、账户相关
- 14篇详细教程
- 支持搜索功能
- 支持浏览次数统计
- 管理端可以CRUD

### 3. 营销活动系统
- 支持多种活动类型：限时抢购、新用户优惠、召回活动、会员日
- 支持多种折扣类型：百分比、固定减免、赠送天数
- 可设置时间范围
- 可设置适用套餐

### 4. 用户分析系统
- DAU/WAU/MAU统计
- 用户留存分析
- 流失预警
- 设备分析

---

## 🎯 下一步

1. **立即**: 重启服务测试新功能
2. **今天**: 完成所有功能测试
3. **本周**: 根据使用情况调整知识库内容
4. **未来**: 考虑添加更多高级功能

---

## 📞 需要帮助？

如果遇到问题，请检查：
1. 服务器日志
2. 浏览器控制台
3. 数据库文件权限
4. 备份文件是否存在

提供以下信息以便排查：
- 错误信息或截图
- 浏览器类型和版本
- 操作步骤
- 服务器日志

---

**迁移完成时间**: 2026-03-02 00:36:04
**状态**: ✅ 成功
**可以开始使用**: ✅ 是
