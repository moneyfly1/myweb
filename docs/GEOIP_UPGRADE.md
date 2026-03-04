# GeoIP 数据库升级完成

## 已完成的改进

### 1. 后端支持多种数据库

已更新 `internal/services/geoip/geoip.go` 以支持：

- **DB-IP City Lite**（推荐）
  - MMDB 格式，与 GeoIP2 兼容
  - 中国城市数据非常详细
  - 完全免费，无需注册
  - 文件大小约 125MB

- **GeoLite2 City**（MaxMind）
  - MMDB 格式
  - 广泛使用的数据库
  - 部分中国 IP 城市数据不完整
  - 文件大小约 60MB

- **IP2Location LITE**
  - BIN 格式
  - 详细的城市、经纬度、邮编、时区信息
  - 需要注册下载

### 2. 自动数据库检测

系统会按优先级自动检测并使用可用的数据库：

1. DB-IP City Lite (dbip-city-lite.mmdb)
2. IP2Location LITE (IP2LOCATION-LITE-DB11.BIN)
3. GeoLite2 City (GeoLite2-City.mmdb)

### 3. 后台管理功能

#### API 端点

- `GET /admin/settings/geoip/status` - 获取数据库状态
  - 显示所有已安装的数据库
  - 显示当前使用的数据库
  - 显示文件大小和更新时间

- `POST /admin/settings/geoip/update` - 下载/更新数据库
  - 支持参数 `type`: "dbip" 或 "geolite2"
  - 自动下载并解压
  - 自动重新加载数据库

#### 前端界面

已更新 `frontend/src/views/admin/Settings.vue`：

- 显示当前使用的数据库
- 显示所有已安装数据库的列表
- 选择要下载的数据库类型（DB-IP 或 GeoLite2）
- 一键下载/更新数据库
- 显示下载进度

### 4. 下载脚本

创建了便捷的下载脚本：

- `scripts/download_dbip.go` - 下载 DB-IP 数据库
- `scripts/download_geoip.go` - 下载 GeoLite2 数据库
- `scripts/download_ip2location.go` - IP2Location 下载说明
- `scripts/test_geoip.sh` - 测试数据库功能

### 5. 测试结果

使用 DB-IP 数据库测试问题 IP：

```
✅ 61.242.235.58 -> 中国, Jinrongjie (Xicheng District)
✅ 182.37.160.182 -> 中国, Zhu Cheng City
✅ 8.8.8.8 -> 美国, Mountain View
✅ 240e:47c:6a0e:e8a0:c5e9:e0e5:e0e5:e0e5 -> 中国, Beijing
```

**之前只显示"中国"的 IP 现在都能显示具体城市了！**

## 使用方法

### 方式1：通过后台管理界面（推荐）

1. 登录管理后台
2. 进入"系统设置" -> "基本设置"
3. 滚动到"GeoIP 数据库管理"部分
4. 选择数据库类型（推荐选择 DB-IP）
5. 点击"下载/更新数据库"按钮
6. 等待下载完成（约 1-2 分钟）

### 方式2：通过命令行

```bash
# 下载 DB-IP 数据库（推荐）
go run scripts/download_dbip.go

# 或下载 GeoLite2 数据库
go run scripts/download_geoip.go

# 测试数据库
bash scripts/test_geoip.sh
```

### 方式3：手动下载

1. 访问 https://db-ip.com/db/download/ip-to-city-lite
2. 下载 MMDB 格式
3. 保存为 `dbip-city-lite.mmdb` 到项目根目录
4. 重启应用

## 建议

1. **推荐使用 DB-IP City Lite**
   - 对中国 IP 的城市数据最详细
   - 完全免费，无需注册
   - 可以直接通过后台下载

2. **定期更新**
   - 建议每月更新一次数据库
   - 可以通过后台管理界面一键更新

3. **监控状态**
   - 在后台可以查看当前使用的数据库
   - 查看数据库文件大小和更新时间

## 文档

详细文档请参考：
- `docs/GEOIP_DATABASES.md` - 数据库对比和配置指南

## 技术细节

### 数据库优先级

系统会自动按以下顺序查找数据库：

1. `./dbip-city-lite.mmdb`
2. `./data/dbip-city-lite.mmdb`
3. `./IP2LOCATION-LITE-DB11.BIN`
4. `./IP2LOCATION-LITE-DB11.IPV6.BIN`
5. `./GeoLite2-City.mmdb`
6. `./data/GeoLite2-City.mmdb`

### 支持的格式

- **MMDB** - MaxMind 数据库格式（GeoIP2, DB-IP）
- **BIN** - IP2Location 数据库格式

### 自动重载

当通过后台更新数据库时，系统会自动重新加载，无需重启应用。

## 问题解决

如果遇到问题：

1. 检查数据库文件是否存在
2. 检查文件权限
3. 查看应用日志
4. 尝试重新下载数据库

## 总结

现在您的系统已经支持：
- ✅ 多种免费 GeoIP 数据库
- ✅ 详细的中国城市信息
- ✅ 后台一键下载/更新
- ✅ 自动数据库检测和切换
- ✅ 完全免费，无需 API 调用
