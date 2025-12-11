<?php

declare(strict_types=1);

use App\Interfaces\MigrationInterface;
use App\Services\DB;

return new class() implements MigrationInterface {
    public function up(): int
    {
        $pdo = DB::getPdo();

        // 添加节点采集相关字段到 node 表
        try {
            $pdo->exec("
                ALTER TABLE `node` 
                ADD COLUMN IF NOT EXISTS `is_collected` tinyint(1) unsigned NOT NULL DEFAULT 0 COMMENT '是否为采集节点' AFTER `gfw_block`,
                ADD COLUMN IF NOT EXISTS `source_url` varchar(255) NOT NULL DEFAULT '' COMMENT '节点来源URL' AFTER `is_collected`,
                ADD COLUMN IF NOT EXISTS `collected_at` int(11) unsigned NOT NULL DEFAULT 0 COMMENT '采集时间' AFTER `source_url`
            ");
        } catch (\PDOException $e) {
            // 如果字段已存在，忽略错误
            if (str_contains($e->getMessage(), 'Duplicate column name')) {
                // 字段已存在，继续
            } else {
                throw $e;
            }
        }

        // 添加索引
        try {
            $pdo->exec("
                ALTER TABLE `node` 
                ADD INDEX IF NOT EXISTS `idx_collected` (`is_collected`, `collected_at`)
            ");
        } catch (\PDOException $e) {
            // 索引可能已存在，忽略
        }

        return 2025010100;
    }

    public function down(): int
    {
        $pdo = DB::getPdo();

        try {
            $pdo->exec("
                ALTER TABLE `node` 
                DROP INDEX IF EXISTS `idx_collected`,
                DROP COLUMN IF EXISTS `collected_at`,
                DROP COLUMN IF EXISTS `source_url`,
                DROP COLUMN IF EXISTS `is_collected`
            ");
        } catch (\PDOException $e) {
            // 忽略错误
        }

        return 2024061600;
    }
};
