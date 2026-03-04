# 中国城市英文到中文映射表

本文件包含常见中国城市的英文到中文映射，用于 GeoIP 地理位置显示。

## 直辖市
- Beijing -> 北京
- Shanghai -> 上海
- Tianjin -> 天津
- Chongqing -> 重庆

## 省会城市
- Guangzhou -> 广州 (广东)
- Chengdu -> 成都 (四川)
- Hangzhou -> 杭州 (浙江)
- Wuhan -> 武汉 (湖北)
- Xi'an -> 西安 (陕西)
- Nanjing -> 南京 (江苏)
- Shenyang -> 沈阳 (辽宁)
- Harbin -> 哈尔滨 (黑龙江)
- Changchun -> 长春 (吉林)
- Jinan -> 济南 (山东)
- Zhengzhou -> 郑州 (河南)
- Shijiazhuang -> 石家庄 (河北)
- Taiyuan -> 太原 (山西)
- Hohhot -> 呼和浩特 (内蒙古)
- Urumqi -> 乌鲁木齐 (新疆)
- Lanzhou -> 兰州 (甘肃)
- Yinchuan -> 银川 (宁夏)
- Xining -> 西宁 (青海)
- Lhasa -> 拉萨 (西藏)
- Kunming -> 昆明 (云南)
- Guiyang -> 贵阳 (贵州)
- Nanning -> 南宁 (广西)
- Haikou -> 海口 (海南)
- Fuzhou -> 福州 (福建)
- Nanchang -> 南昌 (江西)
- Changsha -> 长沙 (湖南)
- Hefei -> 合肥 (安徽)

## 主要城市
- Shenzhen -> 深圳
- Xiamen -> 厦门
- Qingdao -> 青岛
- Dalian -> 大连
- Suzhou -> 苏州
- Wuxi -> 无锡
- Ningbo -> 宁波
- Wenzhou -> 温州
- Dongguan -> 东莞
- Foshan -> 佛山
- Zhuhai -> 珠海
- Zhongshan -> 中山
- Huizhou -> 惠州

## 特别行政区
- Hong Kong -> 香港
- Kowloon -> 九龙
- Macau -> 澳门

## 如何添加新城市

如果需要添加新的城市映射，请编辑 `internal/services/geoip/geoip.go` 文件中的 `translateCityName` 函数，在 `cityMap` 中添加新的映射关系：

```go
cityMap := map[string]string{
    // ... 现有映射 ...
    "YourCityEN": "你的城市中文名",
}
```

## 省份/地区映射

省份和地区的映射在 `translateRegionName` 函数中定义。

## 翻译逻辑

1. 优先查找完整城市名匹配
2. 移除常见后缀（City, Shi）后再查找
3. 移除括号内容后查找
4. 如果城市无法翻译但省份可以翻译，返回"省份 城市"格式
5. 如果都无法翻译，返回清理后的英文原文

## 测试

运行测试脚本验证翻译功能：

```bash
bash scripts/test_geoip.sh
```
