-- 添加设备位置字段
ALTER TABLE devices ADD COLUMN IF NOT EXISTS location VARCHAR(255);

-- 添加注释
COMMENT ON COLUMN devices.location IS 'GeoIP 位置信息（国家、城市等）';
