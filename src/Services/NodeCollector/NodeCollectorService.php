<?php

declare(strict_types=1);

namespace App\Services\NodeCollector;

use App\Models\Config;
use App\Models\Node;
use App\Services\DB;
use App\Utils\Tools;
use function json_decode;
use function json_encode;
use function time;

/**
 * 节点采集服务 - 协调采集流程
 */
final class NodeCollectorService
{
    private NodeFetcher $fetcher;
    private NodeParser $parser;

    public function __construct()
    {
        $this->fetcher = new NodeFetcher();
        $this->parser = new NodeParser();
    }

    /**
     * 获取采集配置
     *
     * @return array{urls: array<string>, update_interval: int, enable_schedule: bool, filter_keywords: array<string>}
     */
    public function getConfig(): array
    {
        $config = Config::obtain('node_collector_config');
        if (empty($config)) {
            return [
                'urls' => [],
                'update_interval' => 3600,
                'enable_schedule' => false,
                'filter_keywords' => [],
            ];
        }

        $data = json_decode($config, true);
        if (! is_array($data)) {
            return [
                'urls' => [],
                'update_interval' => 3600,
                'enable_schedule' => false,
                'filter_keywords' => [],
            ];
        }

        return [
            'urls' => $data['urls'] ?? [],
            'update_interval' => (int) ($data['update_interval'] ?? 3600),
            'enable_schedule' => (bool) ($data['enable_schedule'] ?? false),
            'filter_keywords' => $data['filter_keywords'] ?? [],
        ];
    }

    /**
     * 保存采集配置
     */
    public function saveConfig(array $config): bool
    {
        try {
            $configJson = json_encode($config);
            $existing = (new Config())->where('item', 'node_collector_config')->first();
            if ($existing !== null) {
                $existing->value = $configJson;
                $existing->save();
            } else {
                // 创建新配置项
                $newConfig = new Config();
                $newConfig->item = 'node_collector_config';
                $newConfig->value = $configJson;
                $newConfig->type = 'array';
                $newConfig->class = 'node_collector';
                $newConfig->is_public = '0';
                $newConfig->save();
            }
            return true;
        } catch (\Exception) {
            return false;
        }
    }

    /**
     * 执行节点采集任务
     *
     * @return array{success: bool, message: string, collected_count: int, updated_count: int}
     */
    public function runUpdateTask(): array
    {
        $config = $this->getConfig();
        $urls = $config['urls'] ?? [];

        if (empty($urls)) {
            return [
                'success' => false,
                'message' => '未配置节点采集源 URL',
                'collected_count' => 0,
                'updated_count' => 0,
            ];
        }

        // 1. 获取节点链接
        $nodeLinks = $this->fetcher->fetch($urls);
        if (empty($nodeLinks)) {
            return [
                'success' => false,
                'message' => '未获取到有效节点',
                'collected_count' => 0,
                'updated_count' => 0,
            ];
        }

        // 2. 解析节点
        $parsedNodes = [];
        foreach ($nodeLinks as $linkData) {
            $nodeData = $this->parser->parse($linkData['url']);
            if ($nodeData !== null) {
                $nodeData['source_url'] = $linkData['source_url'];
                $parsedNodes[] = $nodeData;
            }
        }

        if (empty($parsedNodes)) {
            return [
                'success' => false,
                'message' => '未能解析出有效节点',
                'collected_count' => 0,
                'updated_count' => 0,
            ];
        }

        // 3. 应用过滤关键词
        $filterKeywords = $config['filter_keywords'] ?? [];
        if (! empty($filterKeywords)) {
            $parsedNodes = $this->filterNodes($parsedNodes, $filterKeywords);
        }

        // 4. 保存到数据库
        $result = $this->saveNodesToDatabase($parsedNodes);

        // 5. 更新最后采集时间
        $lastUpdateConfig = (new Config())->where('item', 'node_collector_last_update')->first();
        if ($lastUpdateConfig !== null) {
            $lastUpdateConfig->value = (string) time();
            $lastUpdateConfig->save();
        } else {
            $newConfig = new Config();
            $newConfig->item = 'node_collector_last_update';
            $newConfig->value = (string) time();
            $newConfig->type = 'int';
            $newConfig->class = 'node_collector';
            $newConfig->is_public = '0';
            $newConfig->save();
        }

        return $result;
    }

    /**
     * 过滤节点（根据关键词）
     *
     * @param array<array<string, mixed>> $nodes
     * @param array<string> $keywords
     * @return array<array<string, mixed>>
     */
    private function filterNodes(array $nodes, array $keywords): array
    {
        $filtered = [];
        foreach ($nodes as $node) {
            $nodeName = $node['name'] ?? '';
            $shouldFilter = false;

            foreach ($keywords as $keyword) {
                if (stripos($nodeName, $keyword) !== false) {
                    $shouldFilter = true;
                    break;
                }
            }

            if (! $shouldFilter) {
                $filtered[] = $node;
            }
        }

        return $filtered;
    }

    /**
     * 将节点保存到数据库
     *
     * @param array<array<string, mixed>> $nodes
     * @return array{success: bool, message: string, collected_count: int, updated_count: int}
     */
    private function saveNodesToDatabase(array $nodes): array
    {
        $collectedCount = 0;
        $updatedCount = 0;

        foreach ($nodes as $nodeData) {
            try {
                // 确定节点类型 (sort 字段)
                $sort = $this->mapNodeTypeToSort($nodeData['type'] ?? '');
                if ($sort === null) {
                    continue; // 不支持的协议类型
                }

                // 查找是否已存在相同节点（根据 server + port + sort）
                $server = $nodeData['server'] ?? '';
                $port = (int) ($nodeData['port'] ?? 0);

                if (empty($server) || $port <= 0) {
                    continue;
                }

                $existingNode = Node::where('server', $server)
                    ->where('sort', $sort)
                    ->where('is_collected', 1)
                    ->first();

                if ($existingNode !== null) {
                    // 更新现有节点
                    $existingNode->name = $nodeData['name'] ?? $existingNode->name;
                    $existingNode->source_url = $nodeData['source_url'] ?? '';
                    $existingNode->collected_at = time();
                    $existingNode->custom_config = json_encode($nodeData);
                    $existingNode->save();
                    $updatedCount++;
                } else {
                    // 创建新节点
                    $newNode = new Node();
                    $newNode->name = $nodeData['name'] ?? "Node-{$server}:{$port}";
                    $newNode->server = $server;
                    $newNode->sort = $sort;
                    $newNode->type = 1; // 启用
                    $newNode->is_collected = 1;
                    $newNode->source_url = $nodeData['source_url'] ?? '';
                    $newNode->collected_at = time();
                    $newNode->traffic_rate = 1.0;
                    $newNode->node_class = 0;
                    $newNode->node_group = 0;
                    $newNode->password = Tools::genRandomChar(32);
                    $newNode->custom_config = json_encode($nodeData);
                    $newNode->save();
                    $collectedCount++;
                }
            } catch (\Exception) {
                // 跳过错误的节点，继续处理其他节点
                continue;
            }
        }

        return [
            'success' => true,
            'message' => "采集完成：新增 {$collectedCount} 个节点，更新 {$updatedCount} 个节点",
            'collected_count' => $collectedCount,
            'updated_count' => $updatedCount,
        ];
    }

    /**
     * 将节点类型映射到数据库 sort 字段
     *
     * @return int|null 节点类型对应的 sort 值，不支持返回 null
     */
    private function mapNodeTypeToSort(string $type): ?int
    {
        return match ($type) {
            'ss' => 0,        // Shadowsocks
            'ss2022' => 1,    // Shadowsocks2022 (需要特殊处理)
            'tuic' => 2,      // TUIC
            'wireguard' => 3, // WireGuard
            'vmess' => 11,    // Vmess
            'trojan' => 14,   // Trojan
            default => null,
        };
    }

    /**
     * 获取采集状态
     *
     * @return array{is_running: bool, last_update: int|null, next_update: int|null}
     */
    public function getStatus(): array
    {
        $lastUpdate = Config::obtain('node_collector_last_update');
        $config = $this->getConfig();

        $lastUpdateTime = $lastUpdate !== null ? (int) $lastUpdate : null;
        $nextUpdateTime = null;

        if ($lastUpdateTime !== null && $config['enable_schedule']) {
            $nextUpdateTime = $lastUpdateTime + $config['update_interval'];
        }

        return [
            'is_running' => false, // 可以通过其他方式跟踪运行状态
            'last_update' => $lastUpdateTime,
            'next_update' => $nextUpdateTime,
        ];
    }

    /**
     * 获取采集日志（最近 N 条）
     *
     * @return array<array{time: int, level: string, message: string}>
     */
    public function getLogs(int $limit = 100): array
    {
        $logsJson = Config::obtain('node_collector_logs');
        if (empty($logsJson)) {
            return [];
        }

        $logs = json_decode($logsJson, true);
        if (! is_array($logs)) {
            return [];
        }

        // 返回最近 N 条
        return array_slice($logs, -$limit);
    }

    /**
     * 添加日志
     */
    public function addLog(string $message, string $level = 'info'): void
    {
        try {
            $logsJson = Config::obtain('node_collector_logs');
            $logs = [];

            if (! empty($logsJson)) {
                $logs = json_decode($logsJson, true);
                if (! is_array($logs)) {
                    $logs = [];
                }
            }

            $logs[] = [
                'time' => time(),
                'level' => $level,
                'message' => $message,
            ];

            // 只保留最近 1000 条日志
            if (count($logs) > 1000) {
                $logs = array_slice($logs, -1000);
            }

            $logsConfig = (new Config())->where('item', 'node_collector_logs')->first();
            if ($logsConfig !== null) {
                $logsConfig->value = json_encode($logs);
                $logsConfig->save();
            } else {
                $newConfig = new Config();
                $newConfig->item = 'node_collector_logs';
                $newConfig->value = json_encode($logs);
                $newConfig->type = 'array';
                $newConfig->class = 'node_collector';
                $newConfig->is_public = '0';
                $newConfig->save();
            }
        } catch (\Exception) {
            // 忽略日志记录错误
        }
    }
}
