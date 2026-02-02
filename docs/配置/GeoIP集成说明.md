# GeoIP 地理位置解析集成说明

## 功能说明

系统已集成 MaxMind GeoLite2 数据库，可以自动解析用户IP地址到地理位置信息（国家、城市等），用于分析用户地区分布。

## 安装 GeoIP 数据库

### 方法 1：手动下载（推荐）

1. 访问 MaxMind 官网：https://dev.maxmind.com/geoip/geoip2/geolite2/
2. 注册账号（免费）
3. 下载 `GeoLite2-City.mmdb` 文件
4. 将文件放置到项目根目录：`./GeoLite2-City.mmdb`

### 方法 2：使用环境变量指定路径

```bash
export GEOIP_DB_PATH="/path/to/GeoLite2-City.mmdb"
```

### 方法 3：使用默认系统路径

系统会自动查找以下路径：
- `./GeoLite2-City.mmdb`
- `./data/GeoLite2-City.mmdb`
- `/usr/share/GeoIP/GeoLite2-City.mmdb`
- `/var/lib/GeoIP/GeoLite2-City.mmdb`

## 功能特性

### 自动解析

- **登录时**：自动解析登录IP地址的地理位置
- **审计日志**：记录操作时的地理位置信息
- **用户活动**：记录用户活动的地理位置

### 地理位置信息

解析后的地理位置信息包含：
- 国家（中文/英文）
- 国家代码（ISO 3166-1 alpha-2）
- 城市（中文/英文）
- 地区/省份
- 经纬度坐标
- 时区

### 数据格式

地理位置信息以 JSON 格式存储在数据库中：
```json
{
  "country": "中国",
  "country_code": "CN",
  "city": "北京",
  "region": "北京",
  "latitude": 39.9042,
  "longitude": 116.4074,
  "timezone": "Asia/Shanghai"
}
```

## 使用方法

### 1. 分析用户地区分布

运行分析脚本：
```bash
go run scripts/analyze_user_distribution.go
```

脚本会自动：
- 从审计日志中读取已解析的地理位置
- 如果 GeoIP 已启用，对未解析的IP地址进行解析
- 统计用户地区分布
- 显示浏览器、操作系统、设备类型分布
- 分析用户活跃度

### 2. 在代码中使用

```go
import "cboard-go/internal/services/geoip"

// 获取地理位置信息
location, err := geoip.GetLocation("8.8.8.8")
if err == nil {
    fmt.Printf("国家: %s, 城市: %s\n", location.Country, location.City)
}

// 获取格式化的位置字符串（用于数据库存储）
locationStr := geoip.GetLocationString("8.8.8.8")

// 获取简单格式（国家, 城市）
simpleStr := geoip.GetLocationSimple("8.8.8.8")
```

## 注意事项

1. **数据库文件大小**：GeoLite2-City.mmdb 文件约 60-80MB
2. **更新频率**：建议每月更新一次数据库文件以获得最新数据
3. **性能影响**：IP解析速度很快（<1ms），不会影响系统性能
4. **隐私保护**：解析后的地理位置信息仅用于统计分析，不会泄露给第三方

## 故障排除

### GeoIP 未启用

如果看到 "GeoIP 未启用" 的提示：
1. 确认 `GeoLite2-City.mmdb` 文件是否存在
2. 检查文件路径是否正确
3. 确认文件权限（需要读取权限）

### 解析失败

如果某些IP地址解析失败：
- 本地地址（127.0.0.1, ::1）会被跳过
- 内网地址可能无法解析
- 某些新分配的IP地址可能不在数据库中

## 更新数据库

定期更新 GeoIP 数据库以获得最新的地理位置数据：

1. 从 MaxMind 下载最新的 `GeoLite2-City.mmdb`
2. 替换项目中的旧文件
3. 重启服务器（或重新运行分析脚本）

## API 端点

系统会自动在以下场景解析地理位置：
- 用户登录（`/api/v1/auth/login`）
- 审计日志记录（所有需要记录的操作）
- 用户活动记录（如果启用）

地理位置信息会保存在：
- `audit_logs.location` 字段
- `login_history.location` 字段（如果表存在）
- `user_activities.location` 字段

