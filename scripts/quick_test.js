// 专线节点快速测试脚本
// 在浏览器控制台中运行此脚本进行测试

// 配置信息
const CONFIG = {
    cloudflare_api_key: '<CLOUDFLARE_API_KEY_REMOVED>',
    cloudflare_email: '<CLOUDFLARE_EMAIL_REMOVED>',
    cert_email: '<CERT_EMAIL_REMOVED>'
};

// API基础URL
const API_BASE = '/api/v1';

// 获取token
function getToken() {
    return localStorage.getItem('admin_token') || 
           sessionStorage.getItem('admin_token') || 
           document.cookie.match(/admin_token=([^;]+)/)?.[1];
}

// API调用函数
async function apiCall(method, endpoint, data = null) {
    const token = getToken();
    if (!token) {
        throw new Error('未找到管理员token，请先登录');
    }

    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        }
    };

    if (data) {
        options.body = JSON.stringify(data);
    }

    const response = await fetch(`${API_BASE}${endpoint}`, options);
    const result = await response.json();
    
    return { response, result };
}

// 测试1: 保存系统设置
async function testSaveSettings() {
    console.log('📝 测试1: 保存专线节点系统设置...');
    
    const configs = [
        { key: 'cloudflare_api_key', value: CONFIG.cloudflare_api_key, category: 'custom_node', type: 'string', display_name: 'Cloudflare API Key' },
        { key: 'cloudflare_email', value: CONFIG.cloudflare_email, category: 'custom_node', type: 'string', display_name: 'Cloudflare邮箱' },
        { key: 'cert_email', value: CONFIG.cert_email, category: 'custom_node', type: 'string', display_name: '证书申请邮箱' }
    ];

    for (const config of configs) {
        try {
            // 先尝试更新
            const { result } = await apiCall('PUT', `/admin/configs/${config.key}`, config);
            if (result.success) {
                console.log(`✅ 配置 ${config.key} 保存成功`);
            } else {
                // 尝试创建
                const { result: createResult } = await apiCall('POST', '/admin/configs', config);
                if (createResult.success) {
                    console.log(`✅ 配置 ${config.key} 创建成功`);
                } else {
                    console.error(`❌ 配置 ${config.key} 保存失败:`, createResult);
                }
            }
        } catch (error) {
            console.error(`❌ 配置 ${config.key} 保存出错:`, error);
        }
    }
}

// 测试2: 获取服务器列表
async function testGetServers() {
    console.log('📋 测试2: 获取服务器列表...');
    
    try {
        const { result } = await apiCall('GET', '/admin/servers');
        if (result.success) {
            console.log(`✅ 获取服务器列表成功，共 ${result.data.length} 台服务器`);
            console.table(result.data.map(s => ({
                id: s.id,
                name: s.name,
                host: s.host,
                status: s.status,
                xrayr_installed: s.xrayr_installed || false
            })));
            return result.data;
        } else {
            console.error('❌ 获取服务器列表失败:', result);
            return [];
        }
    } catch (error) {
        console.error('❌ 获取服务器列表出错:', error);
        return [];
    }
}

// 测试3: 测试服务器连接
async function testServerConnection(serverId) {
    console.log(`🔌 测试3: 测试服务器连接 (ID: ${serverId})...`);
    
    try {
        const { result } = await apiCall('POST', `/admin/servers/${serverId}/test`, {});
        if (result.success) {
            console.log('✅ 服务器连接测试成功');
            console.log(result);
            return true;
        } else {
            console.error('❌ 服务器连接测试失败:', result);
            return false;
        }
    } catch (error) {
        console.error('❌ 服务器连接测试出错:', error);
        return false;
    }
}

// 测试4: 自动设置XrayR
async function testAutoSetupXrayR(serverId) {
    console.log(`🚀 测试4: 自动设置XrayR (服务器ID: ${serverId})...`);
    console.log('⏳ 这可能需要几分钟时间，请耐心等待...');
    
    try {
        const { result } = await apiCall('POST', `/admin/servers/${serverId}/xrayr/auto-setup`, {
            api_port: '10086'
        });
        if (result.success) {
            console.log('✅ XrayR自动设置已开始');
            console.log('⏳ 等待30秒后检查状态...');
            
            // 等待30秒
            await new Promise(resolve => setTimeout(resolve, 30000));
            
            // 检查状态
            const { result: checkResult } = await apiCall('POST', `/admin/servers/${serverId}/xrayr/check`, {});
            if (checkResult.success && checkResult.data.installed) {
                console.log('✅ XrayR已成功安装');
            } else {
                console.warn('⚠️ XrayR可能还在安装中，请稍后再检查');
            }
            
            return true;
        } else {
            console.error('❌ XrayR自动设置启动失败:', result);
            return false;
        }
    } catch (error) {
        console.error('❌ XrayR自动设置出错:', error);
        return false;
    }
}

// 测试5: 获取XrayR配置
async function testGetXrayRConfig(serverId) {
    console.log(`📥 测试5: 获取XrayR配置 (服务器ID: ${serverId})...`);
    
    try {
        const { result } = await apiCall('POST', `/admin/servers/${serverId}/xrayr/config`, {});
        if (result.success) {
            console.log('✅ 获取XrayR配置成功');
            console.log('API地址:', result.data.api_url);
            console.log('API密钥:', result.data.api_key);
            console.log('完整配置:', result.data);
            return result.data;
        } else {
            console.error('❌ 获取XrayR配置失败:', result);
            return null;
        }
    } catch (error) {
        console.error('❌ 获取XrayR配置出错:', error);
        return null;
    }
}

// 测试6: 创建专线节点
async function testCreateCustomNode(serverId) {
    console.log(`➕ 测试6: 创建测试专线节点 (服务器ID: ${serverId})...`);
    
    // 生成随机UUID和端口
    const uuid = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
    const port = Math.floor(Math.random() * (65535 - 10000 + 1)) + 10000;
    
    const nodeData = {
        server_id: serverId,
        name: `测试节点-${Date.now()}`,
        protocol: 'vmess',
        domain: '',
        port: port,
        uuid: uuid,
        network: 'tcp',
        security: 'none',
        traffic_limit: 10,
        expire_time: null,
        follow_user_expire: false
    };
    
    try {
        const { result } = await apiCall('POST', '/admin/custom-nodes', nodeData);
        if (result.success) {
            console.log('✅ 专线节点创建请求已提交');
            console.log('节点ID:', result.data.id);
            console.log('节点信息:', result.data);
            
            // 等待节点创建
            console.log('⏳ 等待30秒后检查节点状态...');
            await new Promise(resolve => setTimeout(resolve, 30000));
            
            // 检查节点状态
            const { result: nodesResult } = await apiCall('GET', '/admin/custom-nodes');
            if (nodesResult.success) {
                const node = nodesResult.data.find(n => n.id === result.data.id);
                if (node) {
                    console.log('节点状态:', node.status);
                    console.log('节点是否激活:', node.is_active);
                }
            }
            
            return result.data.id;
        } else {
            console.error('❌ 创建专线节点失败:', result);
            return null;
        }
    } catch (error) {
        console.error('❌ 创建专线节点出错:', error);
        return null;
    }
}

// 主测试函数
async function runFullTest() {
    console.log('==========================================');
    console.log('🚀 专线节点完整流程测试');
    console.log('==========================================');
    console.log('');
    
    // 检查token
    const token = getToken();
    if (!token) {
        console.error('❌ 未找到管理员token，请先登录管理后台');
        return;
    }
    console.log('✅ 已找到管理员token');
    console.log('');
    
    // 测试1: 保存系统设置
    await testSaveSettings();
    console.log('');
    
    // 测试2: 获取服务器列表
    const servers = await testGetServers();
    console.log('');
    
    if (servers.length === 0) {
        console.warn('⚠️ 没有找到服务器，跳过后续测试');
        return;
    }
    
    // 选择第一个服务器进行测试
    const testServer = servers[0];
    console.log(`📌 使用服务器进行测试: ${testServer.name} (ID: ${testServer.id})`);
    console.log('');
    
    // 测试3: 测试服务器连接
    const connectionOk = await testServerConnection(testServer.id);
    console.log('');
    
    if (!connectionOk) {
        console.warn('⚠️ 服务器连接失败，跳过XrayR测试');
        return;
    }
    
    // 询问是否继续
    const continueTest = confirm('是否继续测试XrayR自动安装？\n（这可能需要几分钟）');
    if (!continueTest) {
        console.log('测试已取消');
        return;
    }
    
    // 测试4: 自动设置XrayR
    const xrayrOk = await testAutoSetupXrayR(testServer.id);
    console.log('');
    
    if (xrayrOk) {
        // 测试5: 获取XrayR配置
        await testGetXrayRConfig(testServer.id);
        console.log('');
        
        // 询问是否创建测试节点
        const createNode = confirm('是否创建测试节点？');
        if (createNode) {
            // 测试6: 创建专线节点
            await testCreateCustomNode(testServer.id);
        }
    }
    
    console.log('');
    console.log('==========================================');
    console.log('✅ 测试完成');
    console.log('==========================================');
}

// 导出测试函数
window.testCustomNode = {
    runFullTest,
    testSaveSettings,
    testGetServers,
    testServerConnection,
    testAutoSetupXrayR,
    testGetXrayRConfig,
    testCreateCustomNode
};

// 自动运行（可选）
console.log('📋 专线节点测试工具已加载');
console.log('使用方法:');
console.log('  1. 运行完整测试: testCustomNode.runFullTest()');
console.log('  2. 单独测试: testCustomNode.testSaveSettings()');
console.log('');
console.log('开始自动运行完整测试...');
runFullTest();


