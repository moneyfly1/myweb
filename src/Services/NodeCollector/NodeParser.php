<?php

declare(strict_types=1);

namespace App\Services\NodeCollector;

use function base64_decode;
use function explode;
use function json_decode;
use function parse_str;
use function preg_match;
use function str_replace;
use function strlen;
use function substr;
use function urldecode;

/**
 * 节点解析器 - 负责解析各种协议的节点链接
 */
final class NodeParser
{
    /**
     * 统一解析入口
     *
     * @return array<string, mixed>|null 解析后的节点配置，失败返回 null
     */
    public static function parse(string $nodeUrl): ?array
    {
        $nodeUrl = trim($nodeUrl);
        if (empty($nodeUrl)) {
            return null;
        }

        try {
            if (str_starts_with($nodeUrl, 'vmess://')) {
                return self::parseVmess($nodeUrl);
            }
            if (str_starts_with($nodeUrl, 'ss://')) {
                return self::parseSS($nodeUrl);
            }
            if (str_starts_with($nodeUrl, 'trojan://')) {
                return self::parseTrojan($nodeUrl);
            }
            if (str_starts_with($nodeUrl, 'vless://')) {
                return self::parseVless($nodeUrl);
            }
            if (str_starts_with($nodeUrl, 'ssr://')) {
                return self::parseSSR($nodeUrl);
            }
            if (str_starts_with($nodeUrl, 'hysteria2://') || str_starts_with($nodeUrl, 'hy2://')) {
                return self::parseHysteria2($nodeUrl);
            }
            if (str_starts_with($nodeUrl, 'tuic://')) {
                return self::parseTuic($nodeUrl);
            }
        } catch (\Exception) {
            return null;
        }

        return null;
    }

    /**
     * URL Safe Base64 解码，自动补全 Padding
     */
    private static function safeBase64Decode(string $s): string
    {
        $s = str_replace(['-', '_'], ['+', '/'], $s);
        $padding = strlen($s) % 4;
        if ($padding !== 0) {
            $s .= str_repeat('=', 4 - $padding);
        }

        $decoded = base64_decode($s, true);
        return $decoded !== false ? $decoded : '';
    }

    /**
     * 解析 VMess 协议
     */
    private static function parseVmess(string $url): ?array
    {
        try {
            $encoded = substr($url, 8);
            $decoded = self::safeBase64Decode($encoded);

            if (empty($decoded)) {
                return null;
            }

            $data = json_decode($decoded, true);
            if (! is_array($data)) {
                return null;
            }

            if (empty($data['add']) || empty($data['port']) || empty($data['id'])) {
                return null;
            }

            $node = [
                'name' => $data['ps'] ?? $data['add'] ?? 'VMess Node',
                'type' => 'vmess',
                'server' => $data['add'],
                'port' => (int) $data['port'],
                'uuid' => $data['id'],
                'alterId' => (int) ($data['aid'] ?? 0),
                'security' => $data['scy'] ?? 'auto',
                'network' => $data['net'] ?? 'tcp',
            ];

            // TLS 配置
            if (($data['tls'] ?? '') === 'tls') {
                $node['tls'] = true;
                $node['sni'] = $data['sni'] ?? $data['add'] ?? '';
            }

            // 网络类型配置
            $network = $data['net'] ?? 'tcp';
            if ($network === 'ws') {
                $node['ws-opts'] = [
                    'path' => $data['path'] ?? '/',
                    'headers' => [
                        'Host' => $data['host'] ?? $data['add'] ?? '',
                    ],
                ];
            } elseif ($network === 'grpc') {
                $node['grpc-opts'] = [
                    'grpc-service-name' => $data['path'] ?? '',
                ];
            }

            return $node;
        } catch (\Exception) {
            return null;
        }
    }

    /**
     * 解析 Shadowsocks 协议
     */
    private static function parseSS(string $url): ?array
    {
        try {
            // 分离名称部分
            $name = null;
            if (str_contains($url, '#')) {
                [$main, $name] = explode('#', $url, 2);
                $name = urldecode($name);
            } else {
                $main = $url;
            }

            $main = substr($main, 5); // 移除 'ss://'

            // 解析 Base64 部分
            if (str_contains($main, '@')) {
                [$base64Part, $serverPort] = explode('@', $main, 2);
                $authInfo = self::safeBase64Decode($base64Part);
            } else {
                // 整个都是 Base64
                $decoded = self::safeBase64Decode($main);
                if (str_contains($decoded, '@')) {
                    [$authInfo, $serverPort] = explode('@', $decoded, 2);
                } else {
                    return null;
                }
            }

            if (empty($serverPort) || ! str_contains($serverPort, ':')) {
                return null;
            }

            [$server, $port] = explode(':', $serverPort, 2);
            $port = (int) $port;

            if (str_contains($authInfo, ':')) {
                [$method, $password] = explode(':', $authInfo, 2);
            } else {
                // SS2022 格式
                $method = $authInfo;
                $password = '';
            }

            return [
                'name' => $name ?? "SS-{$server}:{$port}",
                'type' => 'ss',
                'server' => $server,
                'port' => $port,
                'method' => $method,
                'password' => $password,
            ];
        } catch (\Exception) {
            return null;
        }
    }

    /**
     * 解析 Trojan 协议
     */
    private static function parseTrojan(string $url): ?array
    {
        try {
            $url = substr($url, 9); // 移除 'trojan://'

            $name = null;
            if (str_contains($url, '#')) {
                [$main, $name] = explode('#', $url, 2);
                $name = urldecode($name);
            } else {
                $main = $url;
            }

            // 解析查询参数
            $query = '';
            if (str_contains($main, '?')) {
                [$main, $query] = explode('?', $main, 2);
            }

            if (str_contains($main, '@')) {
                [$password, $serverPort] = explode('@', $main, 2);
            } else {
                return null;
            }

            if (! str_contains($serverPort, ':')) {
                return null;
            }

            [$server, $port] = explode(':', $serverPort, 2);
            $port = (int) $port;

            $node = [
                'name' => $name ?? "Trojan-{$server}:{$port}",
                'type' => 'trojan',
                'server' => $server,
                'port' => $port,
                'password' => $password,
            ];

            // 解析查询参数
            if (! empty($query)) {
                parse_str($query, $params);
                if (isset($params['sni'])) {
                    $node['sni'] = $params['sni'];
                }
            }

            return $node;
        } catch (\Exception) {
            return null;
        }
    }

    /**
     * 解析 VLESS 协议
     */
    private static function parseVless(string $url): ?array
    {
        try {
            $parsed = parse_url($url);
            if ($parsed === false || empty($parsed['host'])) {
                return null;
            }

            $name = $parsed['fragment'] ?? null;
            if ($name !== null) {
                $name = urldecode($name);
            }

            $node = [
                'name' => $name ?? "VLESS-{$parsed['host']}:{$parsed['port']}",
                'type' => 'vless',
                'server' => $parsed['host'],
                'port' => (int) ($parsed['port'] ?? 443),
                'uuid' => $parsed['user'] ?? '',
            ];

            // 解析查询参数
            if (! empty($parsed['query'])) {
                parse_str($parsed['query'], $params);
                if (isset($params['security'])) {
                    $node['security'] = $params['security'];
                }
                if (isset($params['sni'])) {
                    $node['sni'] = $params['sni'];
                }
            }

            return $node;
        } catch (\Exception) {
            return null;
        }
    }

    /**
     * 解析 SSR 协议
     */
    private static function parseSSR(string $url): ?array
    {
        try {
            $encoded = substr($url, 6); // 移除 'ssr://'
            $decoded = self::safeBase64Decode($encoded);

            if (empty($decoded)) {
                return null;
            }

            // SSR 格式: server:port:protocol:method:obfs:password_base64/?obfsparam=...&protoparam=...
            if (str_contains($decoded, '/?')) {
                [$serverPart, $paramsPart] = explode('/?', $decoded, 2);
            } else {
                $serverPart = $decoded;
                $paramsPart = '';
            }

            $parts = explode(':', $serverPart);
            if (count($parts) < 6) {
                return null;
            }

            [$server, $port, $protocol, $method, $obfs, $passwordBase64] = $parts;
            $password = self::safeBase64Decode($passwordBase64);

            $node = [
                'name' => "SSR-{$server}:{$port}",
                'type' => 'ssr',
                'server' => $server,
                'port' => (int) $port,
                'protocol' => $protocol,
                'method' => $method,
                'obfs' => $obfs,
                'password' => $password,
            ];

            // 解析参数
            if (! empty($paramsPart)) {
                parse_str($paramsPart, $params);
                if (isset($params['remarks'])) {
                    $node['name'] = urldecode(self::safeBase64Decode($params['remarks']));
                }
            }

            return $node;
        } catch (\Exception) {
            return null;
        }
    }

    /**
     * 解析 Hysteria2 协议
     */
    private static function parseHysteria2(string $url): ?array
    {
        try {
            $url = str_replace(['hysteria2://', 'hy2://'], '', $url);
            $parsed = parse_url('hy2://' . $url);

            if ($parsed === false || empty($parsed['host'])) {
                return null;
            }

            $name = $parsed['fragment'] ?? null;
            if ($name !== null) {
                $name = urldecode($name);
            }

            $node = [
                'name' => $name ?? "Hysteria2-{$parsed['host']}:{$parsed['port']}",
                'type' => 'hysteria2',
                'server' => $parsed['host'],
                'port' => (int) ($parsed['port'] ?? 443),
                'password' => $parsed['user'] ?? '',
            ];

            return $node;
        } catch (\Exception) {
            return null;
        }
    }

    /**
     * 解析 TUIC 协议
     */
    private static function parseTuic(string $url): ?array
    {
        try {
            $url = substr($url, 6); // 移除 'tuic://'
            $parsed = parse_url('tuic://' . $url);

            if ($parsed === false || empty($parsed['host'])) {
                return null;
            }

            $name = $parsed['fragment'] ?? null;
            if ($name !== null) {
                $name = urldecode($name);
            }

            $node = [
                'name' => $name ?? "TUIC-{$parsed['host']}:{$parsed['port']}",
                'type' => 'tuic',
                'server' => $parsed['host'],
                'port' => (int) ($parsed['port'] ?? 443),
                'password' => $parsed['user'] ?? '',
                'uuid' => $parsed['pass'] ?? '',
            ];

            // 解析查询参数
            if (! empty($parsed['query'])) {
                parse_str($parsed['query'], $params);
                if (isset($params['sni'])) {
                    $node['sni'] = $params['sni'];
                }
            }

            return $node;
        } catch (\Exception) {
            return null;
        }
    }
}
