@echo off
REM 数据库迁移脚本 - Windows版本
REM 使用方法: migrate_new_features.bat your_database.db

setlocal enabledelayedexpansion

if "%~1"=="" (
    echo 错误: 请提供数据库文件路径
    echo 使用方法: %~nx0 ^<数据库文件路径^>
    echo 示例: %~nx0 C:\path\to\your\cboard.db
    exit /b 1
)

set "DB_FILE=%~1"

if not exist "%DB_FILE%" (
    echo 错误: 数据库文件不存在: %DB_FILE%
    exit /b 1
)

echo === 开始迁移新增功能到数据库 ===
echo 目标数据库: %DB_FILE%
echo.

REM 备份数据库
set "BACKUP_FILE=%DB_FILE%.backup.%date:~0,4%%date:~5,2%%date:~8,2%_%time:~0,2%%time:~3,2%%time:~6,2%"
set "BACKUP_FILE=%BACKUP_FILE: =0%"
echo 1. 备份数据库...
copy "%DB_FILE%" "%BACKUP_FILE%" >nul
echo    √ 备份完成: %BACKUP_FILE%
echo.

echo 2. 创建新表结构...

REM 创建签到记录表
sqlite3 "%DB_FILE%" "CREATE TABLE IF NOT EXISTS checkin_records (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, amount DECIMAL(10,2) NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE INDEX IF NOT EXISTS idx_checkin_user_id ON checkin_records(user_id); CREATE INDEX IF NOT EXISTS idx_checkin_created_at ON checkin_records(created_at);"
echo    √ checkin_records 表创建完成

REM 创建知识库分类表
sqlite3 "%DB_FILE%" "CREATE TABLE IF NOT EXISTS knowledge_categories (id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(100) NOT NULL, icon VARCHAR(50), sort_order INTEGER DEFAULT 0, is_active BOOLEAN DEFAULT 1, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);"
echo    √ knowledge_categories 表创建完成

REM 创建知识库文章表
sqlite3 "%DB_FILE%" "CREATE TABLE IF NOT EXISTS knowledge_articles (id INTEGER PRIMARY KEY AUTOINCREMENT, category_id INTEGER NOT NULL, title VARCHAR(200) NOT NULL, content TEXT NOT NULL, summary VARCHAR(500), view_count INTEGER DEFAULT 0, sort_order INTEGER DEFAULT 0, is_active BOOLEAN DEFAULT 1, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (category_id) REFERENCES knowledge_categories(id)); CREATE INDEX IF NOT EXISTS idx_knowledge_category_id ON knowledge_articles(category_id);"
echo    √ knowledge_articles 表创建完成

REM 创建营销活动表
sqlite3 "%DB_FILE%" "CREATE TABLE IF NOT EXISTS promotions (id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(100) NOT NULL, type VARCHAR(30) NOT NULL, discount_type VARCHAR(20) NOT NULL, discount_value DECIMAL(10,2) NOT NULL, min_amount DECIMAL(10,2) DEFAULT 0, max_discount DECIMAL(10,2) DEFAULT 0, package_ids VARCHAR(500), start_time DATETIME NOT NULL, end_time DATETIME NOT NULL, is_active BOOLEAN DEFAULT 1, description TEXT, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE INDEX IF NOT EXISTS idx_promotion_type ON promotions(type);"
echo    √ promotions 表创建完成
echo.

echo 3. 更新 users 表...
REM 注意: SQLite的ALTER TABLE ADD COLUMN在Windows上可能需要特殊处理
sqlite3 "%DB_FILE%" "ALTER TABLE users ADD COLUMN telegram_id INTEGER;" 2>nul
sqlite3 "%DB_FILE%" "ALTER TABLE users ADD COLUMN telegram_username VARCHAR(100);" 2>nul
sqlite3 "%DB_FILE%" "CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);" 2>nul
echo    √ users 表更新完成
echo.

echo 4. 导入知识库初始数据...
if exist "scripts\init_knowledge.sql" (
    sqlite3 "%DB_FILE%" < scripts\init_knowledge.sql
    echo    √ 知识库初始数据导入完成
) else (
    echo    警告: scripts\init_knowledge.sql 文件不存在，跳过数据导入
)
echo.

echo 5. 更新客户端教程...
if exist "scripts\update_knowledge_tutorials.sql" (
    sqlite3 "%DB_FILE%" < scripts\update_knowledge_tutorials.sql
    echo    √ 教程更新完成
) else (
    echo    警告: scripts\update_knowledge_tutorials.sql 文件不存在，跳过教程更新
)
echo.

echo 6. 验证迁移结果...
for /f %%i in ('sqlite3 "%DB_FILE%" "SELECT COUNT(*) FROM checkin_records;"') do set "COUNT_CHECKIN=%%i"
for /f %%i in ('sqlite3 "%DB_FILE%" "SELECT COUNT(*) FROM knowledge_categories;"') do set "COUNT_CAT=%%i"
for /f %%i in ('sqlite3 "%DB_FILE%" "SELECT COUNT(*) FROM knowledge_articles;"') do set "COUNT_ART=%%i"
for /f %%i in ('sqlite3 "%DB_FILE%" "SELECT COUNT(*) FROM promotions;"') do set "COUNT_PROMO=%%i"

echo 表统计:
echo   - checkin_records: %COUNT_CHECKIN% 条记录
echo   - knowledge_categories: %COUNT_CAT% 条记录
echo   - knowledge_articles: %COUNT_ART% 条记录
echo   - promotions: %COUNT_PROMO% 条记录
echo.

echo === 迁移完成 ===
echo.
echo 备份文件: %BACKUP_FILE%
echo 如果迁移出现问题，可以手动恢复备份文件
echo.
echo 下一步:
echo 1. 重启后端服务: go run cmd\server\main.go
echo 2. 访问前端页面测试新功能
echo 3. 检查日志确认无错误
echo.

pause
