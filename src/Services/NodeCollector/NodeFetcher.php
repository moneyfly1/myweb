<?php

declare(strict_types=1);

namespace App\Services\NodeCollector;

use GuzzleHttp\Client;
use GuzzleHttp\Exception\GuzzleException;
use function base64_decode;
use function preg_match_all;
use function str_replace;
use function trim;

/**
 * 节点获取器 - 负责从 URL 下载节点数据
 */
final class NodeFetcher
{
    private Client $httpClient;

    public function __construct()
    {
        $this->httpClient = new Client([
            'timeout' => 30,
            'headers' => [
                'User-Agent' => 'SSPanel-UIM-NodeCollector/1.0',
            ],
        ]);
    }

    /**
     * 从多个 URL 获取节点链接
     *
     * @param array<string> $urls 节点源 URL 列表
     * @return array<array{url: string, source_url: string, source_index: int}>
     */
    public function fetch(array $urls): array
    {
        $nodes = [];

        foreach ($urls as $index => $url) {
            $url = trim($url);
            if (empty($url)) {
                continue;
            }

            try {
                $response = $this->httpClient->get($url);
                $content = (string) $response->getBody();

                // 尝试 Base64 解码
                $decodedContent = $this->tryBase64Decode($content);
                if ($decodedContent !== $content) {
                    $content = $decodedContent;
                }

                // 提取节点链接
                $nodeLinks = $this->extractNodeLinks($content);

                foreach ($nodeLinks as $link) {
                    $nodes[] = [
                        'url' => $link,
                        'source_url' => $url,
                        'source_index' => $index,
                    ];
                }
            } catch (GuzzleException $e) {
                // 记录错误但继续处理其他 URL
                continue;
            }
        }

        return $nodes;
    }

    /**
     * 尝试 Base64 解码，兼容 URL 安全格式和无填充
     */
    private function tryBase64Decode(string $text): string
    {
        try {
            $cleanText = str_replace([' ', "\n", "\r", "\t"], '', $text);
            $cleanText = str_replace(['-', '_'], ['+', '/'], $cleanText);

            // 补全 padding
            $padding = strlen($cleanText) % 4;
            if ($padding !== 0) {
                $cleanText .= str_repeat('=', 4 - $padding);
            }

            $decoded = base64_decode($cleanText, true);
            if ($decoded === false) {
                return $text;
            }

            return $decoded;
        } catch (\Exception) {
            return $text;
        }
    }

    /**
     * 从内容中提取节点链接
     *
     * @return array<string>
     */
    private function extractNodeLinks(string $content): array
    {
        $links = [];

        // 匹配所有已知协议
        // 协议列表: vmess|ss|ssr|trojan|vless|hysteria2|hy2|hysteria|hy|tuic
        $pattern = '/(?:vmess|ss|ssr|trojan|vless|hysteria2|hy2|hysteria|hy|tuic):\/\/[^\s\n<>"]+/i';
        preg_match_all($pattern, $content, $matches);

        if (! empty($matches[0])) {
            $seen = [];
            foreach ($matches[0] as $link) {
                // 简单验证 URL 结构
                if (strlen($link) < 15) {
                    continue;
                }

                $parts = explode('://', $link);
                if (count($parts) < 2) {
                    continue;
                }

                // 去重
                if (! isset($seen[$link])) {
                    $seen[$link] = true;
                    $links[] = $link;
                }
            }
        }

        return $links;
    }
}
