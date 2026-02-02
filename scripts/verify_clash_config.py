#!/usr/bin/env python3
"""
验证 Clash 配置模板的有效性
"""

import yaml
import sys
from pathlib import Path

def verify_clash_config(config_path):
    """验证 Clash 配置文件"""
    try:
        # 读取配置文件
        with open(config_path, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)
        
        print("✅ YAML 配置解析成功!")
        print()
        
        # 检查基本配置
        print("📋 基本配置:")
        print(f"  - 端口: {config.get('port')}")
        print(f"  - SOCKS 端口: {config.get('socks-port')}")
        print(f"  - 模式: {config.get('mode')}")
        print(f"  - 日志级别: {config.get('log-level')}")
        print()
        
        # 检查 DNS 配置
        dns_config = config.get('dns', {})
        print("🌐 DNS 配置:")
        print(f"  - 启用: {dns_config.get('enable')}")
        print(f"  - 增强模式: {dns_config.get('enhanced-mode')}")
        print(f"  - 国内 DNS: {len(dns_config.get('nameserver', []))} 个")
        print(f"  - 国际 DNS: {len(dns_config.get('fallback', []))} 个")
        print()
        
        # 检查代理组
        proxy_groups = config.get('proxy-groups', [])
        print(f"👥 代理组配置: {len(proxy_groups)} 个")
        for group in proxy_groups:
            name = group.get('name', '未命名')
            group_type = group.get('type', 'unknown')
            proxies_count = len(group.get('proxies', []))
            print(f"  - {name}")
            print(f"    类型: {group_type}, 选项数: {proxies_count}")
        print()
        
        # 检查规则
        rules = config.get('rules', [])
        print(f"📜 分流规则: {len(rules)} 条")
        
        # 统计规则类型
        rule_types = {}
        rule_targets = {}
        for rule in rules:
            if isinstance(rule, str):
                parts = rule.split(',')
                if len(parts) >= 2:
                    rule_type = parts[0]
                    target = parts[-1] if len(parts) >= 3 else 'UNKNOWN'
                    rule_types[rule_type] = rule_types.get(rule_type, 0) + 1
                    rule_targets[target] = rule_targets.get(target, 0) + 1
        
        print("\n  规则类型统计:")
        for rule_type, count in sorted(rule_types.items(), key=lambda x: x[1], reverse=True):
            print(f"    - {rule_type}: {count} 条")
        
        print("\n  分流目标统计:")
        for target, count in sorted(rule_targets.items(), key=lambda x: x[1], reverse=True)[:10]:
            print(f"    - {target}: {count} 条")
        print()
        
        # 验证配置完整性
        print("🔍 配置完整性检查:")
        issues = []
        
        if not config.get('port'):
            issues.append("  ⚠️  缺少端口配置")
        
        if not config.get('mode'):
            issues.append("  ⚠️  缺少运行模式配置")
        
        if not proxy_groups:
            issues.append("  ⚠️  没有配置代理组")
        
        if not rules:
            issues.append("  ⚠️  没有配置规则")
        
        # 检查是否有 MATCH 规则（应该在最后）
        has_match = False
        if rules:
            last_rule = rules[-1]
            if isinstance(last_rule, str) and last_rule.startswith('MATCH,'):
                has_match = True
        
        if not has_match:
            issues.append("  ⚠️  建议在规则末尾添加 MATCH 规则")
        
        if issues:
            for issue in issues:
                print(issue)
        else:
            print("  ✅ 所有检查项通过!")
        
        print()
        print("=" * 60)
        print("验证完成! 配置文件格式正确。")
        return True
        
    except yaml.YAMLError as e:
        print(f"❌ YAML 解析错误: {e}")
        return False
    except FileNotFoundError:
        print(f"❌ 文件不存在: {config_path}")
        return False
    except Exception as e:
        print(f"❌ 验证失败: {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == '__main__':
    # 获取配置文件路径
    if len(sys.argv) > 1:
        config_path = sys.argv[1]
    else:
        # 默认路径
        config_path = Path(__file__).parent.parent / 'uploads' / 'config' / 'temp.yaml'
    
    print(f"验证配置文件: {config_path}")
    print("=" * 60)
    print()
    
    success = verify_clash_config(config_path)
    sys.exit(0 if success else 1)
