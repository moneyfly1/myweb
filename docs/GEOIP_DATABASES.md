# GeoIP 数据库配置指南

本系统支持多种免费的 IP 地理位置数据库，可以自动检测并使用。

## 支持的数据库

### 1. DB-IP City Lite（推荐）✅

**优点：**
- 完全免费，无需注册
- 城市级别数据详细，特别是中国地区
- 每月更新
- MMDB 格式，与 GeoIP2 兼容
- 可以直接下载

**下载方式：**
```bash
go run scripts/download_dbip.go
```

**手动下载：**
访问 https://db-ip.com/db/download/ip-to-city-lite
下载 MMDB 格式，保存为 `dbip-city-lite.mmdb`

**文件位置：**
- `./dbip-city-lite.mmdb`
- `./data/dbip-city-lite.mmdb`

---

### 2. GeoLite2 City（MaxMind）

**优点：**
- 广泛使用
- 数据质量高
- MMDB 格式

**缺点：**
- 中国部分 IP 段城市数据不完整
- 需要注册账号下载（免费）

**下载方式：**
```bash
go run scripts/download_geoip.go
```

**文件位置：**
- `./GeoLite2-City.mmdb`
- `./data/GeoLite2-City.mmdb`

---

### 3. IP2Location LITE DB11

**优点：**
- 包含详细的城市、经纬度、邮编、时区信息
- 支持 IPv4 和 IPv6
- 完全免费

**缺点：**
- 需要注册账号下载
- BIN 格式，需要专门的库

**下载方式：**
1. 访问 https://lite.ip2location.com
2. 注册免费账号
3. 下载 DB11.LITE (包含国家、地区、城市、经纬度、邮编、时区)
4. 解压 ZIP 文件
5. 将 .BIN 文件放到项目根目录

**文件位置：**
- `./IP2LOCATION-LITE-DB11.BIN` (IPv4)
- `./IP2LOCATION-LITE-DB11.IPV6.BIN` (IPv6)
- `./data/IP2LOCATION-LITE-DB11.BIN`
- `./data/IP2LOCATION-LITE-DB11.IPV6.BIN`

---

## 数据库优先级

系统会按以下顺序自动查找并使用数据库：

1. DB-IP City Lite (dbip-city-lite.mmdb)
2. IP2Location LITE (IP2LOCATION-LITE-DB11.BIN)
3. GeoLite2 City (GeoLite2-City.mmdb)

找到第一个可用的数据库后即停止查找。

---

## 使用方法

### 自动检测

系统启动时会自动检测并加载可用的数据库，无需配置。

### 手动指定

如果需要指定特定的数据库文件：

```go
import "cboard-go/internal/services/geoip"

// 初始化时指定数据库路径
err := geoip.InitGeoIP("/path/to/database.mmdb")
```

---

## 测试数据库

下载数据库后，可以测试效果：

```bash
# 测试 DB-IP
go run scripts/download_dbip.go

# 测试 GeoLite2
go run scripts/download_geoip.go
```

---

## 数据库对比

| 数据库 | 格式 | 中国城市数据 | 下载方式 | 更新频率 |
|--------|------|--------------|----------|----------|
| DB-IP City Lite | MMDB | ⭐⭐⭐⭐⭐ 详细 | 直接下载 | 每月 |
| GeoLite2 City | MMDB | ⭐⭐⭐ 一般 | 需注册 | 每周 |
| IP2Location LITE | BIN | ⭐⭐⭐⭐ 详细 | 需注册 | 每月 |

---

## 常见问题

**Q: 为什么有些 IP 只显示国家，没有城市？**

A: 这是因为免费数据库对某些 IP 段的城市级别数据不完整。建议使用 DB-IP City Lite，它对中国地区的数据更详细。

**Q: 可以同时使用多个数据库吗？**

A: 系统会自动选择第一个可用的数据库。如果需要切换，删除当前数据库文件，系统会自动使用下一个可用的。

**Q: 数据库需要多久更新一次？**

A: 建议每月更新一次，以获取最新的 IP 地址分配信息。

**Q: 数据库文件很大，会影响性能吗？**

A: 不会。数据库文件在启动时加载到内存，查询速度非常快（微秒级别）。

---

## 推荐配置

对于中国用户为主的应用，推荐使用 **DB-IP City Lite**：

```bash
# 下载 DB-IP 数据库
go run scripts/download_dbip.go

# 重启应用
# 系统会自动检测并使用 DB-IP 数据库
```

这样可以获得最详细的中国城市信息，完全免费且无需注册。
