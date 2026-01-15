<template>
  <div class="list-container help-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>帮助文档</h1>
      <p>使用指南和常见问题解答</p>
    </div>

    <div class="help-content">
      <!-- 快速导航 -->
      <el-card class="nav-card">
        <template #header>
          <div class="card-header">
            <i class="el-icon-menu"></i>
            快速导航
          </div>
        </template>
        
        <div class="nav-links">
          <el-button 
            v-for="section in sections" 
            :key="section.id"
            type="primary" 
            plain
            @click="scrollToSection(section.id)"
          >
            {{ section.title }}
          </el-button>
        </div>
      </el-card>

      <!-- 使用指南 -->
      <el-card class="guide-card" id="guide">
        <template #header>
          <div class="card-header">
            <i class="el-icon-guide"></i>
            使用指南
          </div>
        </template>

        <el-collapse v-model="activeNames">
          <el-collapse-item 
            v-for="guide in guides" 
            :key="guide.id"
            :title="guide.title"
            :name="guide.id"
          >
            <div class="guide-content" v-html="sanitizeHtml(guide.content)"></div>
          </el-collapse-item>
        </el-collapse>
      </el-card>

      <!-- 常见问题 -->
      <el-card class="faq-card" id="faq">
        <template #header>
          <div class="card-header">
            <i class="el-icon-question"></i>
            常见问题
          </div>
        </template>

        <el-collapse v-model="activeFAQ">
          <el-collapse-item 
            v-for="faq in faqs" 
            :key="faq.id"
            :title="faq.question"
            :name="faq.id"
          >
            <div class="faq-content" v-html="sanitizeHtml(faq.answer)"></div>
          </el-collapse-item>
        </el-collapse>
      </el-card>

      <!-- 客户端下载 -->
      <el-card class="clients-card" id="clients">
        <template #header>
          <div class="card-header">
            <i class="el-icon-download"></i>
            客户端下载
          </div>
        </template>

        <div class="clients-grid">
          <div 
            v-for="client in clients" 
            :key="client.id"
            class="client-item"
          >
            <div class="client-icon">
              <i :class="client.icon"></i>
            </div>
            <div class="client-info">
              <h4>{{ client.name }}</h4>
              <p>{{ client.description }}</p>
              <div class="client-platforms">
                <el-tag 
                  v-for="platform in client.platforms" 
                  :key="platform"
                  size="small"
                  style="margin-right: 5px;"
                >
                  {{ platform }}
                </el-tag>
              </div>
            </div>
            <div class="client-actions">
              <el-button 
                type="primary" 
                size="small"
                @click="downloadClient(client)"
              >
                下载
              </el-button>
              <el-button 
                type="info" 
                size="small"
                @click="viewClientGuide(client)"
              >
                教程
              </el-button>
            </div>
          </div>
        </div>
      </el-card>

      <!-- 客户端教程对话框 -->
      <el-dialog
        v-model="showGuideDialog"
        :title="currentGuideClient?.name + ' 使用教程'"
        width="80%"
        :close-on-click-modal="false"
        class="client-guide-dialog"
      >
        <div class="guide-dialog-content" v-html="sanitizeHtml(currentGuideContent)"></div>
        <template #footer>
          <el-button @click="showGuideDialog = false">关闭</el-button>
          <el-button type="primary" @click="downloadClient(currentGuideClient)">
            前往下载
          </el-button>
        </template>
      </el-dialog>

      <!-- 联系我们 -->
      <el-card class="contact-card" id="contact">
        <template #header>
          <div class="card-header">
            <i class="el-icon-service"></i>
            联系我们
          </div>
        </template>

        <div class="contact-info">
          <div class="contact-item" v-if="contactEmail">
            <i class="el-icon-message"></i>
            <div class="contact-details">
              <h4>售后邮箱</h4>
              <p>{{ contactEmail }}</p>
            </div>
          </div>
          
          <div class="contact-item" v-if="contactQQ">
            <i class="el-icon-chat-dot-round"></i>
            <div class="contact-details">
              <h4>售后QQ</h4>
              <p>{{ contactQQ }}</p>
            </div>
          </div>
          
          <div class="contact-item">
            <i class="el-icon-time"></i>
            <div class="contact-details">
              <h4>服务时间</h4>
              <p>周一至周日 9:00-22:00</p>
            </div>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import DOMPurify from 'dompurify'

export default {
  name: 'Help',
  setup() {
    // HTML内容清理函数，防止XSS攻击
    const sanitizeHtml = (html) => {
      if (!html) return ''
      return DOMPurify.sanitize(html, {
        ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'b', 'i', 'u', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'a', 'div', 'span', 'blockquote', 'pre', 'code'],
        ALLOWED_ATTR: ['href', 'target', 'style', 'class', 'id'],
        ALLOW_DATA_ATTR: false
      })
    }
    const activeNames = ref(['guide-1'])
    const activeFAQ = ref(['faq-1'])
    const showGuideDialog = ref(false)
    const currentGuideClient = ref(null)
    const currentGuideContent = ref('')
    
    // 动态获取联系信息
    const contactEmail = ref('')
    const contactQQ = ref('')
    
    // 从API获取联系信息
    const loadContactInfo = async () => {
      try {
        const api = (await import('@/utils/api')).default
        const response = await api.get('/settings/public-settings')
        if (response && response.data) {
          // 处理不同的响应格式
          let settings = null
          if (response.data.success !== false) {
            // 如果响应有 data 字段，使用 data；否则直接使用 response.data
            settings = response.data.data || response.data
          } else {
            settings = response.data
          }
          
          if (settings) {
            // 处理售后邮箱
            if (settings.support_email !== undefined && settings.support_email !== null) {
              const email = String(settings.support_email).trim()
              if (email !== '') {
                contactEmail.value = email
              }
            }
            
            // 处理售后QQ
            if (settings.support_qq !== undefined && settings.support_qq !== null) {
              const qq = String(settings.support_qq).trim()
              if (qq !== '') {
                contactQQ.value = qq
              }
            }
          }
        }
      } catch (error) {
        console.error('获取联系信息失败:', error)
        // 不设置默认值，如果获取失败则为空
      }
    }
    
    // 组件挂载时加载联系信息
    onMounted(() => {
      loadContactInfo()
    })

    const sections = [
      { id: 'guide', title: '使用指南' },
      { id: 'faq', title: '常见问题' },
      { id: 'clients', title: '客户端下载' },
      { id: 'contact', title: '联系我们' }
    ]

    const guides = [
      {
        id: 'guide-1',
        title: '如何注册账户？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>注册步骤：</strong></p>
            <ol>
              <li>点击页面右上角的"注册"按钮</li>
              <li>输入您的邮箱地址（建议使用常用邮箱，用于接收验证邮件和重要通知）</li>
              <li>设置密码（不少于8位，建议包含字母和数字，提高账户安全性）</li>
              <li>确认密码（确保两次输入的密码一致）</li>
              <li>点击"注册"按钮提交注册信息</li>
              <li>查收邮箱验证邮件（如未收到，请检查垃圾邮件文件夹）</li>
              <li>点击邮件中的验证链接完成邮箱验证</li>
              <li>验证成功后返回登录页面，使用注册的邮箱和密码登录</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>邮箱验证是必须的，未验证的账户无法使用订阅服务。如果长时间未收到验证邮件，请联系客服。
            </p>
          </div>
        `
      },
      {
        id: 'guide-2',
        title: '如何购买套餐？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>购买流程：</strong></p>
            <ol>
              <li>登录账户后，点击导航栏中的"套餐订阅"</li>
              <li>浏览可用的套餐列表，查看每个套餐的流量、时长、价格等信息</li>
              <li>选择适合您需求的套餐（可根据使用场景选择不同流量和时长的套餐）</li>
              <li>点击"立即购买"按钮</li>
              <li>选择支付方式（支持支付宝、微信支付等）</li>
              <li>使用手机扫描二维码完成支付</li>
              <li>支付成功后，系统会自动开通服务并发送通知邮件</li>
              <li>在"仪表板"页面可以查看订阅状态和到期时间</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>套餐购买后立即生效，订阅地址会自动生成。建议在购买前先查看套餐详情，确认流量和时长是否符合需求。
            </p>
          </div>
        `
      },
      {
        id: 'guide-3',
        title: '如何获取订阅地址？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>获取步骤：</strong></p>
            <ol>
              <li>登录账户后，进入"订阅管理"页面</li>
              <li>在页面中找到"订阅地址"区域</li>
              <li>可以看到两种订阅地址：
                <ul style="margin: 10px 0 0 20px;">
                  <li><strong>Clash订阅地址：</strong>适用于Clash系列客户端</li>
                  <li><strong>通用订阅地址：</strong>适用于其他客户端</li>
                </ul>
              </li>
              <li>点击对应地址右侧的"复制"按钮复制订阅地址</li>
              <li>订阅地址已复制到剪贴板，可以直接粘贴到客户端软件中使用</li>
              <li>也可以扫描页面上的二维码，直接导入到Shadowrocket等支持扫码的客户端</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #fff7e6; border-left: 3px solid #faad14; border-radius: 4px;">
              <strong>⚠️ 重要提示：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>订阅地址包含您的账户信息，请勿分享给他人</li>
                <li>每个账户的订阅地址是唯一的</li>
                <li>如果订阅地址泄露，请及时重置订阅地址</li>
                <li>订阅地址重置后，旧地址将立即失效</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 'guide-4',
        title: '如何重置订阅地址？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>重置步骤：</strong></p>
            <ol>
              <li>登录账户后，进入"订阅管理"页面</li>
              <li>在页面底部找到"重置订阅地址"按钮（蓝色按钮）</li>
              <li>点击"重置订阅地址"按钮</li>
              <li>系统会弹出确认对话框，提示"重置订阅地址将清空所有设备记录"</li>
              <li>仔细阅读提示信息，确认是否继续</li>
              <li>点击"确定"按钮确认重置操作（重置后旧地址将立即失效）</li>
              <li>重置成功后，系统会生成新的订阅地址</li>
              <li>复制新的订阅地址，重新配置所有使用该订阅的设备</li>
              <li>更新所有客户端软件的订阅配置</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #fff1f0; border-left: 3px solid #ff4d4f; border-radius: 4px;">
              <strong>⚠️ 注意事项：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>重置后，所有使用旧订阅地址的设备将无法连接</li>
                <li>需要重新配置所有设备（手机、电脑、路由器等）</li>
                <li>建议在重置前记录当前使用的设备列表</li>
                <li>重置操作不可撤销，请谨慎操作</li>
              </ul>
            </p>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 何时需要重置：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>设备数量达到上限时</li>
                <li>订阅地址可能泄露时</li>
                <li>需要清理所有设备连接时</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 'guide-5',
        title: '如何查看设备列表？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>查看步骤：</strong></p>
            <ol>
              <li>登录账户后，点击导航栏中的"设备管理"</li>
              <li>在设备管理页面可以查看所有已连接的设备</li>
              <li>设备列表显示以下信息：
                <ul style="margin: 10px 0 0 20px;">
                  <li>设备名称或标识</li>
                  <li>设备类型（手机、电脑、路由器等）</li>
                  <li>IP地址</li>
                  <li>最后访问时间</li>
                  <li>在线状态</li>
                </ul>
              </li>
              <li>可以点击"移除"按钮删除不需要的设备</li>
              <li>移除设备后，该设备将无法继续使用订阅服务</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>定期检查设备列表，移除不使用的设备，可以释放设备配额，让新设备能够连接。
            </p>
          </div>
        `
      },
      {
        id: 'guide-6',
        title: '如何修改密码？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>修改步骤：</strong></p>
            <ol>
              <li>登录账户后，点击右上角的用户头像或用户名</li>
              <li>进入"个人资料"或"账户设置"页面</li>
              <li>找到"修改密码"区域</li>
              <li>输入当前密码（用于验证身份）</li>
              <li>输入新密码（不少于8位，建议包含字母和数字）</li>
              <li>确认新密码（再次输入新密码确保一致）</li>
              <li>点击"修改密码"或"保存"按钮</li>
              <li>修改成功后，系统会提示您重新登录</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 安全提示：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>定期更换密码可以提高账户安全性</li>
                <li>不要使用过于简单的密码（如123456、password等）</li>
                <li>不要在多个网站使用相同的密码</li>
                <li>如果忘记密码，可以使用"忘记密码"功能重置</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 'guide-7',
        title: '如何充值账户余额？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>充值步骤：</strong></p>
            <ol>
              <li>登录账户后，进入"仪表板"页面</li>
              <li>在"账户余额"卡片中，点击"充值"按钮</li>
              <li>在弹出的充值对话框中，输入充值金额（默认20元，可自定义）</li>
              <li>可以选择快速金额（20、50、100、200、500、1000元）或自定义金额</li>
              <li>点击"确认充值"按钮</li>
              <li>系统会生成支付二维码，使用手机支付宝扫描二维码完成支付</li>
              <li>支付成功后，余额会自动到账，可以用于购买套餐</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>默认充值金额20元，支持自定义任意金额</li>
                <li>充值成功后，余额可用于购买套餐或升级设备数量</li>
                <li>可以在"订单记录"页面查看充值记录</li>
                <li>如果支付后余额未到账，请联系客服处理</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 'guide-8',
        title: '如何升级设备数量？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>升级步骤：</strong></p>
            <ol>
              <li>登录账户后，进入"订阅管理"页面</li>
              <li>在页面中找到"升级设备数量"按钮（仅在订阅有效时显示）</li>
              <li>点击"升级设备数量"按钮</li>
              <li>在弹出的对话框中，选择要升级到的设备数量</li>
              <li>系统会显示升级费用（根据当前设备数量和目标设备数量计算）</li>
              <li>确认升级信息后，点击"确认升级"按钮</li>
              <li>如果账户余额充足，系统会直接从余额扣除费用并完成升级</li>
              <li>如果余额不足，系统会引导您先充值，然后完成升级</li>
              <li>升级成功后，设备数量限制会立即更新</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>升级设备数量需要订阅处于有效状态</li>
                <li>升级费用根据设备数量差异计算</li>
                <li>可以使用账户余额支付，也可以先充值再支付</li>
                <li>升级后立即生效，无需等待</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 'guide-9',
        title: '什么是套餐折算？如何折算？',
        content: `
          <div style="line-height: 1.8;">
            <p><strong>套餐折算说明：</strong></p>
            <p>当您已有套餐，想要购买不同类型的套餐时，系统会将当前套餐的剩余时间折算成余额，返还到您的账户。</p>
            
            <p><strong>折算规则：</strong></p>
            <ul style="margin: 10px 0 0 20px;">
              <li>系统会根据您当前套餐的剩余天数和价值计算折算金额</li>
              <li><strong>折算公式：折算金额 = 剩余天数 × (原套餐价格 ÷ 原套餐天数)</strong></li>
              <li>折算后的金额会返还到您的账户余额</li>
              <li>折算后，当前套餐的设备和时间都会清零</li>
            </ul>
            
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>📐 折算公式详解：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li><strong>剩余天数</strong>：当前套餐到期时间减去当前时间的剩余天数（向上取整）</li>
                <li><strong>原套餐价格</strong>：您购买当前套餐时支付的原价</li>
                <li><strong>原套餐天数</strong>：当前套餐的有效期天数</li>
                <li><strong>每天单价</strong>：原套餐价格 ÷ 原套餐天数</li>
                <li><strong>折算金额</strong>：剩余天数 × 每天单价（保留两位小数）</li>
              </ul>
              <p style="margin-top: 10px;"><strong>示例：</strong></p>
              <p style="margin-left: 20px;">
                假设您购买了一个 30 天、价格 ¥100 的套餐，使用了 10 天后想要折算：<br>
                剩余天数 = 20 天<br>
                每天单价 = ¥100 ÷ 30 = ¥3.33<br>
                折算金额 = 20 × ¥3.33 = ¥66.60
              </p>
            </p>
            
            <p><strong>折算步骤：</strong></p>
            <ol>
              <li>登录账户后，进入"套餐订阅"页面</li>
              <li>选择要购买的新套餐，点击"立即购买"</li>
              <li>如果系统检测到您已有套餐，会弹出折算提示对话框</li>
              <li>对话框会显示：
                <ul style="margin: 10px 0 0 20px;">
                  <li>当前套餐的剩余天数</li>
                  <li>可折算的金额</li>
                  <li>折算后的操作说明</li>
                </ul>
              </li>
              <li>仔细阅读提示信息，确认是否进行折算</li>
              <li>点击"立即折算"按钮确认折算</li>
              <li>系统会将剩余时间折算成余额返还到您的账户</li>
              <li>折算完成后，可以继续购买新套餐</li>
            </ol>
            
            <p style="margin-top: 15px; padding: 10px; background: #fff1f0; border-left: 3px solid #ff4d4f; border-radius: 4px;">
              <strong>⚠️ 重要提示：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>折算后，当前套餐的所有设备和时间都会清零</li>
                <li>需要重新配置所有设备使用新的订阅地址</li>
                <li>折算操作不可撤销，请谨慎操作</li>
                <li>建议在折算前记录当前使用的设备列表</li>
              </ul>
            </p>
            
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 何时需要折算：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>想要从流量套餐切换到时长套餐时</li>
                <li>想要从时长套餐切换到流量套餐时</li>
                <li>想要更换不同类型的套餐时</li>
                <li>如果购买相同类型的套餐，系统会自动累加时间，无需折算</li>
              </ul>
            </p>
          </div>
        `
      }
    ]

    const faqs = [
      {
        id: 'faq-1',
        question: '为什么我的订阅无法使用？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>可能的原因和解决方法：</strong></p>
            <ul>
              <li><strong>订阅已过期：</strong>请检查订阅到期时间，如果已过期需要续费</li>
              <li><strong>设备数量超限：</strong>您的订阅可能已达到最大设备数量限制，请重置订阅地址或移除不使用的设备</li>
              <li><strong>网络连接问题：</strong>请检查您的网络连接是否正常，尝试切换网络环境</li>
              <li><strong>客户端配置错误：</strong>请检查客户端配置是否正确，尝试重新导入订阅</li>
              <li><strong>节点故障：</strong>当前使用的节点可能暂时不可用，请尝试切换其他节点</li>
              <li><strong>订阅地址失效：</strong>如果订阅地址已重置，需要使用新的订阅地址重新配置</li>
            </ul>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 排查步骤：</strong>1. 检查订阅状态和到期时间 2. 查看设备列表是否超限 3. 尝试更新订阅 4. 切换节点测试 5. 重新配置客户端
            </p>
          </div>
        `
      },
      {
        id: 'faq-2',
        question: '如何查看我的设备列表？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>查看步骤：</strong></p>
            <ol>
              <li>登录账户后，点击导航栏中的"设备管理"</li>
              <li>在设备管理页面可以查看所有已连接的设备</li>
            </ol>
            <p><strong>设备信息包括：</strong></p>
            <ul>
              <li>设备名称或标识</li>
              <li>设备类型（手机、电脑、路由器等）</li>
              <li>IP地址</li>
              <li>最后访问时间</li>
              <li>在线状态（是否当前在线）</li>
            </ul>
            <p><strong>设备管理操作：</strong></p>
            <ul>
              <li>可以点击"移除"按钮删除不需要的设备</li>
              <li>移除设备后，该设备将无法继续使用订阅服务</li>
              <li>移除设备可以释放设备配额，让新设备能够连接</li>
            </ul>
          </div>
        `
      },
      {
        id: 'faq-3',
        question: '支持哪些客户端软件？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>我们支持以下主流客户端软件：</strong></p>
            <ul>
              <li><strong>iOS平台：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>Shadowrocket（推荐，功能强大，需在App Store购买）</li>
                  <li>其他iOS客户端请参考App Store上的相关应用</li>
                </ul>
              </li>
              <li><strong>Android平台：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>Clash Meta for Android（推荐，功能强大）</li>
                  <li>V2RayNG（轻量级，简单易用）</li>
                  <li>Hiddify（跨平台支持）</li>
                  <li>FlClash（Flutter开发）</li>
                </ul>
              </li>
              <li><strong>Windows平台：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>Clash Party（推荐，基于Mihomo内核，功能强大）</li>
                  <li>Clash Verge Rev（界面现代化）</li>
                  <li>Sparkle（轻量级，性能优异）</li>
                  <li>Hiddify（跨平台支持）</li>
                  <li>FlClash（Flutter开发）</li>
                  <li>V2rayN（轻量级）</li>
                </ul>
              </li>
              <li><strong>Mac平台：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>Clash Party（推荐，基于Mihomo内核）</li>
                  <li>Clash Verge Rev（界面现代化）</li>
                  <li>Sparkle（轻量级）</li>
                  <li>Hiddify（跨平台支持）</li>
                  <li>FlClash（Flutter开发）</li>
                </ul>
              </li>
              <li><strong>Linux平台：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>Clash Party（推荐）</li>
                  <li>Clash Verge Rev</li>
                </ul>
              </li>
            </ul>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>所有客户端都支持订阅链接导入，具体使用方法请查看各客户端的详细教程。
            </p>
          </div>
        `
      },
      {
        id: 'faq-4',
        question: '如何修改密码？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>修改密码步骤：</strong></p>
            <ol>
              <li>登录账户后，点击右上角的用户头像或用户名</li>
              <li>进入"个人资料"或"账户设置"页面</li>
              <li>找到"修改密码"区域</li>
              <li>输入当前密码（用于验证身份）</li>
              <li>输入新密码（不少于8位，建议包含字母和数字）</li>
              <li>确认新密码（再次输入新密码确保一致）</li>
              <li>点击"修改密码"或"保存"按钮</li>
              <li>修改成功后，系统会提示您重新登录</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #fff7e6; border-left: 3px solid #faad14; border-radius: 4px;">
              <strong>⚠️ 安全提示：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>不要使用过于简单的密码（如123456、password等）</li>
                <li>建议使用包含大小写字母、数字和特殊字符的复杂密码</li>
                <li>不要在多个网站使用相同的密码</li>
                <li>定期更换密码可以提高账户安全性</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 'faq-5',
        question: '忘记密码怎么办？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>找回密码步骤：</strong></p>
            <ol>
              <li>在登录页面，点击"忘记密码？"链接</li>
              <li>输入您注册时使用的邮箱地址</li>
              <li>点击"发送重置邮件"按钮</li>
              <li>查收邮箱中的重置密码邮件（如未收到，请检查垃圾邮件文件夹）</li>
              <li>点击邮件中的重置密码链接</li>
              <li>在重置页面输入新密码（不少于8位）</li>
              <li>确认新密码</li>
              <li>点击"重置密码"按钮完成重置</li>
              <li>使用新密码登录账户</li>
            </ol>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>如果长时间未收到重置邮件，请检查邮箱是否正确，或联系客服协助处理。
            </p>
          </div>
        `
      },
      {
        id: 'faq-6',
        question: '订阅地址可以分享给他人吗？',
        answer: `
          <div style="line-height: 1.8;">
            <p style="padding: 10px; background: #fff1f0; border-left: 3px solid #ff4d4f; border-radius: 4px;">
              <strong>⚠️ 重要提示：订阅地址绝对不能分享给他人！</strong>
            </p>
            <p><strong>原因：</strong></p>
            <ul>
              <li>订阅地址包含您的账户信息和访问凭证</li>
              <li>他人使用您的订阅地址会占用您的设备配额</li>
              <li>可能导致您的设备无法连接</li>
              <li>存在账户安全风险</li>
            </ul>
            <p><strong>如果订阅地址泄露：</strong></p>
            <ol>
              <li>立即登录账户</li>
              <li>进入"订阅管理"页面</li>
              <li>点击"重置订阅地址"按钮</li>
              <li>确认重置操作</li>
              <li>使用新的订阅地址重新配置所有设备</li>
            </ol>
          </div>
        `
      },
      {
        id: 'faq-7',
        question: '如何测试节点速度？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>不同客户端的测试方法：</strong></p>
            <ul>
              <li><strong>Clash Party / Clash Verge Rev：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>在"代理"页面，右键点击节点</li>
                  <li>选择"延迟测试"或"速度测试"</li>
                  <li>等待测试完成，选择延迟最低的节点</li>
                </ul>
              </li>
              <li><strong>Clash Meta for Android：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>在"代理"页面，长按节点</li>
                  <li>选择"延迟测试"</li>
                  <li>或使用"自动选择"功能</li>
                </ul>
              </li>
              <li><strong>Shadowrocket：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>长按节点名称</li>
                  <li>选择"测试连接"</li>
                  <li>查看延迟和速度信息</li>
                </ul>
              </li>
              <li><strong>V2RayN：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>右键点击节点</li>
                  <li>选择"测试真链接延迟"</li>
                  <li>等待测试完成</li>
                </ul>
              </li>
            </ul>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 提示：</strong>建议定期测试节点速度，选择延迟最低、速度最快的节点，以获得最佳使用体验。
            </p>
          </div>
        `
      },
      {
        id: 'faq-8',
        question: '为什么连接后无法访问某些网站？',
        answer: `
          <div style="line-height: 1.8;">
            <p><strong>可能的原因和解决方法：</strong></p>
            <ul>
              <li><strong>规则模式问题：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>某些网站可能被规则判定为直连，但实际需要代理</li>
                  <li>尝试切换到"全局模式"测试</li>
                  <li>如果全局模式可以访问，说明是规则问题</li>
                </ul>
              </li>
              <li><strong>节点问题：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>当前节点可能无法访问该网站</li>
                  <li>尝试切换其他节点</li>
                  <li>选择不同地区的节点测试</li>
                </ul>
              </li>
              <li><strong>DNS问题：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>DNS解析可能有问题</li>
                  <li>在客户端设置中更换DNS服务器（如8.8.8.8、1.1.1.1）</li>
                </ul>
              </li>
              <li><strong>网站本身问题：</strong>
                <ul style="margin: 5px 0 0 20px;">
                  <li>网站可能暂时无法访问</li>
                  <li>尝试直接访问（不使用代理）测试</li>
                </ul>
              </li>
            </ul>
            <p style="margin-top: 15px; padding: 10px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 排查步骤：</strong>1. 切换节点测试 2. 切换代理模式测试 3. 检查DNS设置 4. 查看客户端日志
            </p>
          </div>
        `
      }
    ]

    const clients = [
      {
        id: 1,
        name: 'Clash Party',
        description: 'Windows/Mac/Linux平台功能强大的代理客户端，基于Mihomo内核',
        icon: 'el-icon-monitor',
        platforms: ['Windows', 'Mac', 'Linux'],
        githubKey: 'clash-party',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">Clash Party 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，根据您的系统（Windows/Mac/Linux）和芯片架构自动下载对应版本</li>
              <li>Windows：下载 .exe 安装包，双击运行安装</li>
              <li>Mac：下载 .pkg 或 .dmg 文件，双击打开并按照提示安装</li>
              <li>Linux：下载 .deb 文件，使用包管理器安装（如：sudo dpkg -i *.deb）</li>
              <li>安装完成后，在应用列表中找到 Clash Party</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 Clash Party 软件</li>
              <li>在设置中找到"订阅"或"配置"选项</li>
              <li>点击"添加订阅"或"从URL导入"</li>
              <li>粘贴您的订阅地址</li>
              <li>输入订阅名称（可自定义）</li>
              <li>点击"确定"或"保存"</li>
              <li>等待配置下载完成</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在应用主界面可以看到所有可用节点</li>
              <li>选择一个节点（可以测试延迟选择最快的）</li>
              <li>开启系统代理或TUN模式</li>
              <li>连接成功后，状态会显示为已连接</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">四、规则配置</h4>
            <ol>
              <li>在应用中可以切换代理模式：
                <ul style="margin: 10px 0 0 20px;">
                  <li><strong>规则模式：</strong>根据规则自动选择代理或直连</li>
                  <li><strong>全局模式：</strong>所有流量都走代理</li>
                  <li><strong>直连模式：</strong>所有流量都直连</li>
                </ul>
              </li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">五、更新订阅</h4>
            <ol>
              <li>在订阅设置中，点击"更新"按钮</li>
              <li>等待更新完成，新的节点会自动同步</li>
              <li>建议定期更新订阅以获取最新节点</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"功能</li>
                <li>可以使用"延迟测试"功能选择最快的节点</li>
                <li>定期更新订阅以获取最新节点</li>
                <li>Clash Party 基于 Mihomo 内核，性能优异</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 2,
        name: 'Clash Verge Rev',
        description: 'Windows/Mac/Linux平台优秀的代理客户端，界面现代化',
        icon: 'el-icon-monitor',
        platforms: ['Windows', 'Mac', 'Linux'],
        githubKey: 'clash-verge',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">Clash Verge Rev 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，根据您的系统（Windows/Mac/Linux）和芯片架构自动下载对应版本</li>
              <li>Windows：下载 .exe 安装包，双击运行安装</li>
              <li>Mac：下载 .dmg 文件，双击打开后将 Clash Verge Rev 拖拽到"应用程序"文件夹</li>
              <li>Linux：下载 .deb 或 .rpm 文件，使用包管理器安装</li>
              <li>首次运行可能需要授予权限，按照系统提示操作</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 Clash Verge Rev 应用</li>
              <li>在设置中找到"订阅"或"配置"选项</li>
              <li>点击"添加订阅"或"从URL导入"</li>
              <li>粘贴您的订阅地址</li>
              <li>输入订阅名称（可自定义）</li>
              <li>点击"确定"或"保存"</li>
              <li>等待配置下载完成</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在应用主界面可以看到所有可用节点</li>
              <li>选择一个节点（可以测试延迟选择最快的）</li>
              <li>开启系统代理或TUN模式</li>
              <li>连接成功后，状态会显示为已连接</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">四、规则配置</h4>
            <ol>
              <li>在应用中可以切换代理模式：
                <ul style="margin: 10px 0 0 20px;">
                  <li><strong>规则模式：</strong>根据规则自动选择代理或直连</li>
                  <li><strong>全局模式：</strong>所有流量都走代理</li>
                  <li><strong>直连模式：</strong>所有流量都直连</li>
                </ul>
              </li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">五、更新订阅</h4>
            <ol>
              <li>在订阅设置中，点击"更新"按钮</li>
              <li>等待更新完成，新的节点会自动同步</li>
              <li>建议定期更新订阅以获取最新节点</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"功能</li>
                <li>可以使用"延迟测试"功能选择最快的节点</li>
                <li>定期更新订阅以获取最新节点</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 3,
        name: 'Sparkle',
        description: 'Windows/Mac平台轻量级代理客户端，性能优异',
        icon: 'el-icon-monitor',
        platforms: ['Windows', 'Mac'],
        githubKey: 'sparkle',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">Sparkle 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，根据您的系统（Windows/Mac）和芯片架构自动下载对应版本</li>
              <li>Windows：下载 .exe 安装包，双击运行安装</li>
              <li>Mac：下载 .dmg 文件，双击打开后将 Sparkle 拖拽到"应用程序"文件夹</li>
              <li>首次运行可能需要授予权限，按照系统提示操作</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 Sparkle 应用</li>
              <li>在设置中找到"订阅"或"配置"选项</li>
              <li>点击"添加订阅"或"从URL导入"</li>
              <li>粘贴您的订阅地址</li>
              <li>输入订阅名称（可自定义）</li>
              <li>点击"确定"或"保存"</li>
              <li>等待配置下载完成</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在应用主界面可以看到所有可用节点</li>
              <li>选择一个节点（可以测试延迟选择最快的）</li>
              <li>开启系统代理或TUN模式</li>
              <li>连接成功后，状态会显示为已连接</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">四、规则配置</h4>
            <ol>
              <li>在应用中可以切换代理模式：
                <ul style="margin: 10px 0 0 20px;">
                  <li><strong>规则模式：</strong>根据规则自动选择代理或直连</li>
                  <li><strong>全局模式：</strong>所有流量都走代理</li>
                  <li><strong>直连模式：</strong>所有流量都直连</li>
                </ul>
              </li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">五、更新订阅</h4>
            <ol>
              <li>在订阅设置中，点击"更新"按钮</li>
              <li>等待更新完成，新的节点会自动同步</li>
              <li>建议定期更新订阅以获取最新节点</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"功能</li>
                <li>可以使用"延迟测试"功能选择最快的节点</li>
                <li>定期更新订阅以获取最新节点</li>
                <li>Sparkle 是轻量级客户端，资源占用低</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 4,
        name: 'Hiddify',
        description: '跨平台代理客户端，支持Windows/Mac/Android',
        icon: 'el-icon-monitor',
        platforms: ['Windows', 'Mac', 'Android'],
        githubKey: 'hiddify',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">Hiddify 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，根据您的系统（Windows/Mac/Android）和芯片架构自动下载对应版本</li>
              <li>Windows：下载 .exe 安装包，双击运行安装</li>
              <li>Mac：下载 .dmg 文件，双击打开后将 Hiddify 拖拽到"应用程序"文件夹</li>
              <li>Android：下载 .apk 文件，在手机上安装（需要允许安装未知来源应用）</li>
              <li>首次运行可能需要授予权限，按照系统提示操作</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 Hiddify 应用</li>
              <li>在设置中找到"订阅"或"配置"选项</li>
              <li>点击"添加订阅"或"从URL导入"</li>
              <li>粘贴您的订阅地址</li>
              <li>输入订阅名称（可自定义）</li>
              <li>点击"确定"或"保存"</li>
              <li>等待配置下载完成</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在应用主界面可以看到所有可用节点</li>
              <li>选择一个节点（可以测试延迟选择最快的）</li>
              <li>开启系统代理或VPN模式（Android需要授予VPN权限）</li>
              <li>连接成功后，状态会显示为已连接</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">四、规则配置</h4>
            <ol>
              <li>在应用中可以切换代理模式：
                <ul style="margin: 10px 0 0 20px;">
                  <li><strong>规则模式：</strong>根据规则自动选择代理或直连</li>
                  <li><strong>全局模式：</strong>所有流量都走代理</li>
                  <li><strong>直连模式：</strong>所有流量都直连</li>
                </ul>
              </li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">五、更新订阅</h4>
            <ol>
              <li>在订阅设置中，点击"更新"按钮</li>
              <li>等待更新完成，新的节点会自动同步</li>
              <li>建议定期更新订阅以获取最新节点</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"功能</li>
                <li>可以使用"延迟测试"功能选择最快的节点</li>
                <li>定期更新订阅以获取最新节点</li>
                <li>Hiddify 支持多平台，界面统一，使用方便</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 5,
        name: 'FlClash',
        description: 'Windows/Mac/Android平台Flutter开发的代理客户端',
        icon: 'el-icon-monitor',
        platforms: ['Windows', 'Mac', 'Android'],
        githubKey: 'flclash',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">FlClash 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，根据您的系统（Windows/Mac/Android）和芯片架构自动下载对应版本</li>
              <li>Windows：下载 .exe 安装包，双击运行安装</li>
              <li>Mac：下载 .dmg 文件，双击打开后将 FlClash 拖拽到"应用程序"文件夹</li>
              <li>Android：下载 .apk 文件，在手机上安装（需要允许安装未知来源应用）</li>
              <li>首次运行可能需要授予权限，按照系统提示操作</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 FlClash 应用</li>
              <li>在设置中找到"订阅"或"配置"选项</li>
              <li>点击"添加订阅"或"从URL导入"</li>
              <li>粘贴您的订阅地址</li>
              <li>输入订阅名称（可自定义）</li>
              <li>点击"确定"或"保存"</li>
              <li>等待配置下载完成</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在应用主界面可以看到所有可用节点</li>
              <li>选择一个节点（可以测试延迟选择最快的）</li>
              <li>开启系统代理或TUN模式</li>
              <li>连接成功后，状态会显示为已连接</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"功能</li>
                <li>可以使用"延迟测试"功能选择最快的节点</li>
                <li>定期更新订阅以获取最新节点</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 6,
        name: 'V2rayN',
        description: 'Windows平台轻量级代理客户端，资源占用低',
        icon: 'el-icon-monitor',
        platforms: ['Windows'],
        githubKey: 'v2rayn',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">V2rayN 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，根据您的系统架构（64位/32位）下载对应版本</li>
              <li>下载完成后，解压到任意文件夹（建议解压到非系统盘）</li>
              <li>解压后找到 v2rayN.exe 文件，双击运行</li>
              <li>首次运行会提示选择语言，选择"中文"</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 V2rayN 软件</li>
              <li>点击顶部菜单栏的"订阅" → "订阅设置"</li>
              <li>点击"添加"按钮</li>
              <li>在"备注"输入框中输入订阅名称（可自定义）</li>
              <li>在"地址（URL）"输入框中粘贴您的订阅地址</li>
              <li>点击"确定"按钮保存订阅</li>
              <li>返回主界面，点击"订阅" → "更新订阅"</li>
              <li>等待更新完成，节点会自动出现在服务器列表中</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在服务器列表中，可以看到所有可用的节点</li>
              <li>双击节点名称可以选择该节点</li>
              <li>选中的节点会显示为蓝色背景</li>
              <li>点击系统托盘中的 V2rayN 图标</li>
              <li>选择"Http代理" → "开启Http代理"</li>
              <li>或者选择"系统代理" → "自动配置系统代理"</li>
              <li>连接成功后，系统托盘图标会显示为绿色</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">四、更新订阅</h4>
            <ol>
              <li>点击顶部菜单栏的"订阅" → "更新订阅"</li>
              <li>或者右键点击系统托盘图标，选择"更新订阅"</li>
              <li>等待更新完成，新的节点会自动同步</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"和"自动配置系统代理"功能</li>
                <li>可以使用"测试真链接延迟"功能选择最快的节点</li>
                <li>建议定期更新订阅和软件版本</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 7,
        name: 'V2rayNG',
        description: 'Android平台轻量级代理客户端，简单易用',
        icon: 'el-icon-mobile-phone',
        platforms: ['Android'],
        githubKey: 'v2rayng',
        downloadUrl: '',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">V2rayNG 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>点击下载按钮，下载最新版本的 APK 文件</li>
              <li>下载完成后，在手机上找到下载的 APK 文件</li>
              <li>如果系统提示"禁止安装未知来源应用"，需要先允许安装</li>
              <li>点击 APK 文件开始安装</li>
              <li>安装完成后，在应用列表中找到 V2rayNG</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 V2rayNG 应用</li>
              <li>点击右上角的"+"按钮</li>
              <li>选择"从剪贴板导入"或"订阅设置"</li>
              <li>在订阅设置中，点击"+"添加订阅</li>
              <li>粘贴您的订阅地址</li>
              <li>输入订阅名称（可自定义）</li>
              <li>点击"确定"保存</li>
              <li>返回主界面，点击"更新订阅"</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在主界面可以看到所有可用节点</li>
              <li>点击节点名称可以选择该节点</li>
              <li>点击右下角的开关按钮，开启代理</li>
              <li>首次开启会提示创建VPN连接，点击"确定"</li>
              <li>授予VPN权限（这是Android系统要求）</li>
              <li>连接成功后，开关会显示为绿色</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"开机自启"和"后台运行"功能</li>
                <li>可以使用"延迟测试"功能选择最快的节点</li>
                <li>定期更新订阅以获取最新节点</li>
              </ul>
            </p>
          </div>
        `
      },
      {
        id: 8,
        name: 'Shadowrocket',
        description: 'iOS平台优秀的代理客户端，界面简洁，操作便捷',
        icon: 'el-icon-iphone',
        platforms: ['iOS'],
        githubKey: null,
        downloadUrl: 'https://apps.apple.com/app/shadowrocket/id932747118',
        guideUrl: '#',
        guide: `
          <div style="line-height: 1.8;">
            <h3 style="color: #1677ff; margin-bottom: 15px;">Shadowrocket 使用教程</h3>
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">一、下载和安装</h4>
            <ol>
              <li>Shadowrocket 是付费应用，需要在 App Store 购买（价格约 $2.99）</li>
              <li>打开 App Store，搜索"Shadowrocket"</li>
              <li>点击"获取"或"购买"按钮进行购买和下载</li>
              <li>下载完成后，在主屏幕找到 Shadowrocket 图标</li>
              <li>首次打开需要授予VPN权限</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">二、配置订阅</h4>
            <ol>
              <li>打开 Shadowrocket 应用</li>
              <li>点击右上角的"+"按钮</li>
              <li>选择"类型"为"Subscribe"（订阅）</li>
              <li>在"URL"输入框中粘贴您的订阅地址</li>
              <li>在"备注"输入框中输入配置名称（可自定义）</li>
              <li>点击右上角的"完成"按钮</li>
              <li>应用会自动下载订阅配置</li>
              <li>下载完成后，节点列表会自动出现在"首页"</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">三、选择节点和连接</h4>
            <ol>
              <li>在应用首页，可以看到所有可用的节点</li>
              <li>点击节点名称可以选择该节点</li>
              <li>选中的节点会显示为蓝色</li>
              <li>点击右上角的开关按钮，开启代理</li>
              <li>首次开启会提示添加VPN配置，点击"允许"</li>
              <li>输入设备密码或使用Face ID/Touch ID确认</li>
              <li>连接成功后，开关会显示为绿色，状态栏会显示VPN图标</li>
            </ol>
            
            <h4 style="color: #333; margin-top: 20px; margin-bottom: 10px;">四、更新订阅</h4>
            <ol>
              <li>在"首页"，向下拉刷新订阅</li>
              <li>或者在"订阅"页面，点击订阅右侧的刷新图标</li>
              <li>等待更新完成，新的节点会自动同步</li>
            </ol>
            
            <p style="margin-top: 20px; padding: 15px; background: #f0f9ff; border-left: 3px solid #1677ff; border-radius: 4px;">
              <strong>💡 使用技巧：</strong>
              <ul style="margin: 10px 0 0 20px;">
                <li>建议开启"自动选择"功能，应用会自动选择延迟最低的节点</li>
                <li>可以使用"节点测试"功能批量测试节点速度</li>
                <li>在"设置"中可以开启"后台运行"，保持代理连接</li>
              </ul>
            </p>
          </div>
        `
      }
    ]

    // 滚动到指定区域
    const scrollToSection = (sectionId) => {
      const element = document.getElementById(sectionId)
      if (element) {
        element.scrollIntoView({ behavior: 'smooth' })
      }
    }

    // 下载客户端
    const downloadClient = async (client) => {
      try {
        // 如果是 Shadowrocket，直接打开 App Store
        if (client.name === 'Shadowrocket') {
          window.open('https://apps.apple.com/app/shadowrocket/id932747118', '_blank')
          return
        }
        
        // 如果有 GitHub Key，使用自动获取
        if (client.githubKey) {
          ElMessage.info('正在获取最新下载链接...')
          const { getClientDownloadUrl, getClientReleasesUrl } = await import('@/utils/githubDownload')
          try {
            const downloadUrl = await getClientDownloadUrl(client.githubKey)
            console.log('获取到下载链接:', downloadUrl)
            
            // 检查是否是加速链接
            const isAccelerated = downloadUrl.includes('ghproxy.com') || downloadUrl.includes('ghproxy.net')
            if (isAccelerated) {
              console.log('使用加速下载:', downloadUrl)
            }
            
            // 对于移动端，直接打开链接让浏览器处理下载
            // 对于桌面端，尝试触发下载
            const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
            
            if (isMobile) {
              // 移动端直接打开链接
              window.open(downloadUrl, '_blank')
              ElMessage.success(isAccelerated ? '已打开下载页面（已启用加速）' : '已打开下载页面')
            } else {
              // 桌面端创建下载链接
              const link = document.createElement('a')
              link.href = downloadUrl
              link.download = '' // 让浏览器自动识别文件名
              link.target = '_blank'
              link.rel = 'noopener noreferrer'
              link.style.display = 'none'
              document.body.appendChild(link)
              link.click()
              
              // 延迟移除，确保下载已开始
              setTimeout(() => {
                if (document.body.contains(link)) {
                  document.body.removeChild(link)
                }
              }, 200)
              
              ElMessage.success(isAccelerated ? '开始下载（已启用加速）...' : '开始下载...')
            }
          } catch (error) {
            console.error('获取下载链接失败:', error)
            ElMessage.error('获取下载链接失败: ' + (error.message || '未知错误'))
            // 备用：打开 releases 页面
            try {
              const { getClientReleasesUrl } = await import('@/utils/githubDownload')
              const releasesUrl = getClientReleasesUrl(client.githubKey)
              if (releasesUrl) {
                // 延迟一下再打开，避免消息被覆盖
                setTimeout(() => {
                  window.open(releasesUrl, '_blank')
                  ElMessage.warning('已打开发布页面，请手动选择下载')
                }, 1000)
              } else {
                ElMessage.error('无法获取下载链接')
              }
            } catch (err) {
              console.error('获取发布页面失败:', err)
              ElMessage.error('无法获取下载链接，请稍后重试')
            }
          }
        } else if (client.downloadUrl) {
          // 如果有直接配置的下载链接
          const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
          
          if (isMobile) {
            window.open(client.downloadUrl, '_blank')
            ElMessage.success('已打开下载页面')
          } else {
            const link = document.createElement('a')
            link.href = client.downloadUrl
            link.download = ''
            link.target = '_blank'
            link.rel = 'noopener noreferrer'
            link.style.display = 'none'
            document.body.appendChild(link)
            link.click()
            setTimeout(() => {
              if (document.body.contains(link)) {
                document.body.removeChild(link)
              }
            }, 200)
            ElMessage.success('开始下载...')
          }
        } else {
          ElMessage.error('下载链接未配置')
        }
      } catch (error) {
        console.error('下载失败:', error)
        ElMessage.error('下载失败: ' + (error.message || '请稍后重试'))
      }
    }

    // 查看客户端教程
    const viewClientGuide = (client) => {
      if (client && client.guide) {
        currentGuideClient.value = client
        currentGuideContent.value = sanitizeHtml(client.guide)
        showGuideDialog.value = true
      } else if (client && client.guideUrl && client.guideUrl !== '#') {
        window.open(client.guideUrl, '_blank')
      }
    }

    // 创建清理后的computed属性
    const sanitizedGuides = computed(() => {
      return guides.map(guide => ({
        ...guide,
        content: sanitizeHtml(guide.content)
      }))
    })

    const sanitizedFaqs = computed(() => {
      return faqs.map(faq => ({
        ...faq,
        answer: sanitizeHtml(faq.answer)
      }))
    })

    return {
      activeNames,
      activeFAQ,
      sections,
      guides: sanitizedGuides,
      faqs: sanitizedFaqs,
      clients,
      contactEmail,
      contactQQ,
      showGuideDialog,
      currentGuideClient,
      currentGuideContent,
      scrollToSection,
      downloadClient,
      viewClientGuide,
      sanitizeHtml
    }
  }
}
</script>

<style scoped lang="scss">
/* 覆盖 list-container 的 padding，移除顶部空隙 */
:deep(.list-container) {
  @media (max-width: 768px) {
    padding-top: 0 !important;
    margin-top: 0 !important;
  }
}

.help-container {
  /* 使用 list-common.scss 的统一样式 */
  /* padding, max-width, margin 由 list-common.scss 统一管理 */
  padding: 0;
  max-width: none;
  margin: 0;
  width: 100%;
  
  /* 移动端：移除顶部padding，让内容紧贴导航栏 */
  @media (max-width: 768px) {
    padding-top: 0 !important;
    margin-top: 0 !important;
  }
}

.page-header {
  margin-bottom: 1rem;
  text-align: left;
  
  /* 移动端：移除上边距，让内容紧贴导航栏 */
  @media (max-width: 768px) {
    margin-top: 0 !important;
    padding-top: 0 !important;
    margin-bottom: 0.75rem;
  }
}

.page-header :is(h1) {
  color: #1677ff;
  font-size: 1.5rem;
  margin-bottom: 0.25rem;
  
  @media (max-width: 768px) {
    font-size: 1.25rem;
  }
}

.page-header :is(p) {
  color: #666;
  font-size: 0.875rem;
  
  @media (max-width: 768px) {
    font-size: 0.8125rem;
  }
}

.help-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  
  @media (max-width: 768px) {
    gap: 0.75rem;
  }
}

.nav-card,
.guide-card,
.faq-card,
.clients-card,
.contact-card {
  border-radius: 8px;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.05);
  
  :deep(.el-card__header) {
    padding: 12px 16px;
    font-size: 0.9375rem;
  }
  
  :deep(.el-card__body) {
    padding: 12px 16px;
  }
  
  @media (max-width: 768px) {
    :deep(.el-card__header) {
      padding: 10px 12px;
      font-size: 0.875rem;
    }
    
    :deep(.el-card__body) {
      padding: 10px 12px;
    }
  }
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  font-size: 0.9375rem;
  
  @media (max-width: 768px) {
    font-size: 0.875rem;
    gap: 0.375rem;
  }
}

.nav-links {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  
  @media (max-width: 768px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 6px;
  }
  
  @media (max-width: 480px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 6px;
  }
  
  :deep(.el-button) {
    width: 100%;
    padding: 10px 16px;
    border-radius: 8px;
    font-weight: 600;
    font-size: 0.875rem;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    
    @media (max-width: 768px) {
      padding: 10px 12px;
      font-size: 0.8125rem;
      border-radius: 8px;
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
      
      &:active {
        transform: scale(0.98);
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
      }
    }
    
    @media (max-width: 480px) {
      padding: 8px 10px;
      font-size: 0.75rem;
      border-radius: 6px;
    }
  }
}

.guide-content,
.faq-content {
  line-height: 1.6;
  color: #333;
}

.guide-content :is(ol),
.faq-content :is(ol) {
  padding-left: 1.25rem;
  margin: 0.5rem 0;
}

.guide-content :is(ul),
.faq-content :is(ul) {
  padding-left: 1.25rem;
  margin: 0.5rem 0;
}

.guide-content :is(li),
.faq-content :is(li) {
  margin-bottom: 0.375rem;
  font-size: 0.875rem;
  line-height: 1.5;
}

.clients-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 0.75rem;
  
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}

.client-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  transition: all 0.3s ease;
  
  @media (max-width: 768px) {
    padding: 0.625rem;
    gap: 0.5rem;
  }
}

.client-item:hover {
  background: #e3f2fd;
  transform: translateY(-2px);
}

.client-icon {
  font-size: 1.5rem;
  color: #1677ff;
  width: 40px;
  text-align: center;
  
  @media (max-width: 768px) {
    font-size: 1.25rem;
    width: 35px;
  }
}

.client-info {
  flex: 1;
}

.client-info h4 {
  margin: 0 0 0.25rem 0;
  color: #333;
  font-weight: 600;
  font-size: 0.9375rem;
  
  @media (max-width: 768px) {
    font-size: 0.875rem;
  }
}

.client-info :is(p) {
  margin: 0 0 0.25rem 0;
  color: #666;
  font-size: 0.8125rem;
  
  @media (max-width: 768px) {
    font-size: 0.75rem;
  }
}

.client-platforms {
  margin-top: 0.25rem;
}

.client-actions {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  
  @media (max-width: 768px) {
    gap: 0.25rem;
  }
}

.contact-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
  
  @media (max-width: 768px) {
    gap: 0.5rem;
  }
}

.contact-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: #f8f9fa;
  border-radius: 6px;
  
  @media (max-width: 768px) {
    padding: 0.625rem;
    gap: 0.5rem;
  }
}

.contact-item :is(i) {
  font-size: 1.5rem;
  color: #1677ff;
  width: 40px;
  text-align: center;
  
  @media (max-width: 768px) {
    font-size: 1.25rem;
    width: 35px;
  }
}

.contact-details h4 {
  margin: 0 0 0.25rem 0;
  color: #333;
  font-weight: 600;
  font-size: 0.9375rem;
  
  @media (max-width: 768px) {
    font-size: 0.875rem;
  }
}

.contact-details :is(p) {
  margin: 0;
  color: #666;
  font-size: 0.8125rem;
  
  @media (max-width: 768px) {
    font-size: 0.75rem;
  }
}

@media (max-width: 768px) {
  .help-container {
    padding: 12px;
  }
  
  .page-header {
    margin-bottom: 1rem;
    padding: 0 4px;
    
    :is(h1) {
      font-size: 1.5rem;
      margin-bottom: 0.5rem;
    }
    
    :is(p) {
      font-size: 0.875rem;
      line-height: 1.5;
    }
  }
  
  .help-content {
    gap: 1rem;
  }
  
  .nav-card,
  .guide-card,
  .faq-card,
  .clients-card,
  .contact-card {
    margin-bottom: 0.75rem;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    
    :deep(.el-card__header) {
      padding: 14px 16px;
      font-size: 0.9375rem;
      border-bottom: 1px solid #f0f0f0;
    }
    
    :deep(.el-card__body) {
      padding: 16px;
    }
  }
  
  .card-header {
    font-size: 0.9375rem;
    font-weight: 600;
    
    :is(i) {
      font-size: 1.1rem;
    }
  }
  
  .nav-links {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
    
    @media (max-width: 480px) {
      grid-template-columns: 1fr; /* 极小屏幕单列 */
    }
    
    :deep(.el-button) {
      width: 100%;
      padding: 12px 16px;
      font-size: 0.875rem;
      border-radius: 8px;
      font-weight: 500;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      margin: 0; /* 移除按钮默认边距 */
      
      &:active {
        transform: scale(0.97);
      }
    }
  }
  
  .guide-content,
  .faq-content {
    font-size: 0.875rem;
    line-height: 1.7;
    
    :is(ol), :is(ul) {
      padding-left: 1.25rem;
      margin: 0.75rem 0;
    }
    
    :is(li) {
      margin-bottom: 0.5rem;
      line-height: 1.6;
    }
    
    :is(p) {
      margin: 0.75rem 0;
      line-height: 1.7;
    }
    
    h3, h4 {
      font-size: 1rem;
      margin-top: 1rem;
      margin-bottom: 0.75rem;
    }
    
    /* 代码块样式优化 */
    :is(pre) {
      background: #f5f7fa;
      padding: 10px;
      border-radius: 6px;
      overflow-x: auto;
      font-family: monospace;
      font-size: 0.85em;
      border: 1px solid #ebeef5;
      margin: 0.75rem 0;
    }
    
    :is(code) {
      background: #f0f2f5;
      padding: 2px 4px;
      border-radius: 4px;
      color: #e6a23c;
      font-family: monospace;
    }
  }
  
  .clients-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  
  .client-item {
    flex-direction: row; /* 改回行布局，图标在左侧 */
    align-items: flex-start;
    text-align: left;
    padding: 16px;
    border-radius: 12px;
    background: #ffffff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    
    &:active {
      transform: scale(0.98);
      box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
    }
  }
  
  .client-icon {
    font-size: 1.75rem;
    width: 40px;
    margin-bottom: 0;
    margin-right: 12px;
    flex-shrink: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    height: 40px; /* 保持图标区域正方形 */
  }
  
  .client-info {
    width: auto;
    flex: 1;
    margin-bottom: 0;
    
    :is(h4) {
      font-size: 1rem;
      margin-bottom: 4px;
      font-weight: 600;
    }
    
    :is(p) {
      font-size: 0.8125rem;
      line-height: 1.4;
      margin-bottom: 6px;
      color: #666;
      display: -webkit-box;
      -webkit-line-clamp: 2; /* 限制描述行数 */
      line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  }
  
  .client-platforms {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-start; /* 左对齐 */
    gap: 6px;
    margin-top: 4px;
    
    :deep(.el-tag) {
      font-size: 0.75rem;
      padding: 0 6px;
      height: 22px;
      line-height: 20px;
      border-radius: 4px;
    }
  }
  
  .client-actions {
    flex-direction: column; /* 按钮垂直排列 */
    justify-content: center;
    gap: 8px;
    width: auto;
    margin-top: 0;
    margin-left: 8px;
    flex-shrink: 0;
    
    :deep(.el-button) {
      flex: none;
      width: 70px;
      padding: 6px 0; /* 减小内边距 */
      font-size: 0.8125rem;
      border-radius: 6px;
      margin: 0;
      height: 28px;
      
      &:active {
        transform: scale(0.95);
      }
    }
  }
  
  .contact-info {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  
  .contact-item {
    flex-direction: column;
    text-align: center;
    padding: 16px;
    border-radius: 12px;
    background: #ffffff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    
    :is(i) {
      font-size: 2rem;
      width: auto;
      margin-bottom: 12px;
    }
  }
  
  .contact-details {
    width: 100%;
    
    :is(h4) {
      font-size: 1rem;
      margin-bottom: 8px;
      font-weight: 600;
    }
    
    :is(p) {
      font-size: 0.875rem;
      line-height: 1.5;
      color: #666;
    }
  }
  
  // 折叠面板优化
  :deep(.el-collapse) {
    border: none;
    
    .el-collapse-item {
      border-bottom: 1px solid #f0f0f0;
      margin-bottom: 8px;
      
      &:last-child {
        border-bottom: none;
        margin-bottom: 0;
      }
    }
    
    .el-collapse-item__header {
      padding: 12px 0;
      font-size: 0.9375rem;
      font-weight: 500;
      color: #333;
    }
    
    .el-collapse-item__content {
      padding: 12px 0 16px 0;
    }
  }
}

// 客户端教程对话框样式
:deep(.client-guide-dialog) {
  @media (max-width: 768px) {
    .el-dialog {
      width: 95% !important;
      margin: 5vh auto !important;
      max-height: 90vh;
    }
    
    .el-dialog__header {
      padding: 16px;
      border-bottom: 1px solid #f0f0f0;
      
      .el-dialog__title {
        font-size: 1.125rem;
        font-weight: 600;
      }
    }
    
    .el-dialog__body {
      max-height: calc(90vh - 120px);
      overflow-y: auto;
      padding: 16px;
      -webkit-overflow-scrolling: touch;
    }
    
    .el-dialog__footer {
      padding: 12px 16px;
      border-top: 1px solid #f0f0f0;
      
      .el-button {
        padding: 10px 20px;
        font-size: 0.875rem;
      }
    }
  }
  
  .guide-dialog-content {
    line-height: 1.8;
    color: #333;
    
    :is(h3) {
      color: #1677ff;
      margin-bottom: 15px;
      font-size: 1.25rem;
      font-weight: 600;
    }
    
    :is(h4) {
      color: #333;
      margin-top: 20px;
      margin-bottom: 10px;
      font-size: 1.1rem;
      font-weight: 600;
    }
    
    :is(ol), :is(ul) {
      padding-left: 1.5rem;
      margin: 10px 0;
      
      :is(li) {
        margin-bottom: 8px;
        line-height: 1.6;
      }
    }
    
    :is(p) {
      margin: 10px 0;
    }
    
    @media (max-width: 768px) {
      font-size: 0.875rem;
      line-height: 1.7;
      
      :is(h3) {
        font-size: 1.125rem;
        margin-bottom: 12px;
      }
      
      :is(h4) {
        font-size: 1rem;
        margin-top: 16px;
        margin-bottom: 8px;
      }
      
      :is(ol), :is(ul) {
        padding-left: 1.5rem;
        margin: 8px 0;
        
        :is(li) {
          margin-bottom: 6px;
          line-height: 1.6;
        }
      }
      
      :is(p) {
        margin: 8px 0;
        line-height: 1.7;
      }
    }
  }
}
</style> 