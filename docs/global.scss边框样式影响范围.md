# global.scss 边框样式修改影响范围

## 修改内容

在 `frontend/src/styles/global.scss` 第 1163-1187 行，修改了表单输入框的边框样式：

**修改前：**
- `.el-input__inner` 有边框（导致"框中框"问题）

**修改后：**
- `.el-input__wrapper` 有边框（单层边框）
- `.el-input__inner` 无边框、背景透明

## 影响范围

### 全局影响
这个修改会影响**所有使用 `.el-form` 内的 `.el-input` 和 `.el-select` 组件**的页面。

### 受影响的主要文件

#### 1. 认证相关页面（用户端）
- ✅ `frontend/src/views/UnifiedAuth.vue` - 统一认证页面（登录/注册/忘记密码）
- ✅ `frontend/src/views/Login.vue` - 登录页面
- ✅ `frontend/src/views/Register.vue` - 注册页面
- ✅ `frontend/src/views/ForgotPassword.vue` - 忘记密码页面

#### 2. 用户设置页面
- ✅ `frontend/src/views/UserSettings.vue` - 用户设置页面
- ✅ `frontend/src/views/Profile.vue` - 用户资料页面

#### 3. 管理员后台页面
- ✅ `frontend/src/views/admin/Settings.vue` - 系统设置
- ✅ `frontend/src/views/admin/Users.vue` - 用户管理（搜索表单）
- ✅ `frontend/src/views/admin/Nodes.vue` - 节点管理
- ✅ `frontend/src/views/admin/Packages.vue` - 套餐管理
- ✅ `frontend/src/views/admin/Orders.vue` - 订单管理
- ✅ `frontend/src/views/admin/Subscriptions.vue` - 订阅管理
- ✅ `frontend/src/views/admin/CustomNodes.vue` - 自定义节点
- ✅ `frontend/src/views/admin/PaymentConfig.vue` - 支付配置
- ✅ `frontend/src/views/admin/Config.vue` - 配置管理
- ✅ `frontend/src/views/admin/ConfigUpdate.vue` - 配置更新
- ✅ `frontend/src/views/admin/Coupons.vue` - 优惠券管理
- ✅ `frontend/src/views/admin/Invites.vue` - 邀请管理
- ✅ `frontend/src/views/admin/UserLevels.vue` - 用户等级管理
- ✅ `frontend/src/views/admin/AbnormalUsers.vue` - 异常用户
- ✅ `frontend/src/views/admin/Statistics.vue` - 统计分析
- ✅ `frontend/src/views/admin/SystemLogs.vue` - 系统日志
- ✅ `frontend/src/views/admin/EmailQueue.vue` - 邮件队列
- ✅ `frontend/src/views/admin/EmailDetail.vue` - 邮件详情
- ✅ `frontend/src/views/admin/Tickets.vue` - 工单管理
- ✅ `frontend/src/views/admin/Profile.vue` - 管理员资料
- ✅ `frontend/src/views/admin/Dashboard.vue` - 管理员仪表盘

#### 4. 组件文件
- ✅ `frontend/src/components/admin/users/UserFormDialog.vue` - 用户表单对话框（已单独处理）
- ✅ `frontend/src/components/admin/users/UserDetailDialog.vue` - 用户详情对话框
- ✅ `frontend/src/components/PaymentForm.vue` - 支付表单
- ✅ `frontend/src/components/ThemeSettings.vue` - 主题设置

#### 5. 其他用户端页面
- ✅ `frontend/src/views/Dashboard.vue` - 用户仪表盘
- ✅ `frontend/src/views/Packages.vue` - 套餐页面
- ✅ `frontend/src/views/Subscription.vue` - 订阅页面
- ✅ `frontend/src/views/Nodes.vue` - 节点页面
- ✅ `frontend/src/views/Orders.vue` - 订单页面
- ✅ `frontend/src/views/Tickets.vue` - 工单页面
- ✅ `frontend/src/views/Invites.vue` - 邀请页面
- ✅ `frontend/src/views/Devices.vue` - 设备页面
- ✅ `frontend/src/views/LoginHistory.vue` - 登录历史

## 已单独处理的文件

以下文件已经有自己的样式覆盖，不受全局样式影响：

1. **`frontend/src/components/admin/users/UserFormDialog.vue`**
   - 有完整的样式覆盖（第 292-639 行）
   - 使用 `:deep()` 选择器覆盖全局样式
   - 确保单层边框效果

2. **`frontend/src/views/admin/Users.vue`**
   - 搜索表单有自定义样式（第 1110-1152 行）
   - 已处理输入框样式

## 影响效果

### 正面影响
✅ 所有表单输入框现在都是**单层边框**，不再有"框中框"问题
✅ 统一的视觉风格
✅ 更好的用户体验

### 需要注意
⚠️ 如果某些页面有自定义的输入框样式，可能需要检查是否需要调整
⚠️ 聚焦效果现在在 `.el-input__wrapper` 上，而不是 `.el-input__inner` 上

## 建议检查的文件

如果发现某些页面的输入框样式异常，可以检查以下文件是否有自定义样式覆盖：

1. `frontend/src/styles/list-common.scss` - 列表通用样式
2. 各个页面的 `<style scoped>` 部分
3. 是否有使用 `!important` 覆盖全局样式的地方

## 测试建议

建议测试以下关键页面：
1. ✅ 统一认证页面（登录/注册）
2. ✅ 用户设置页面
3. ✅ 管理员后台设置页面
4. ✅ 用户管理页面（搜索表单）
5. ✅ 节点管理页面（表单）
6. ✅ 套餐管理页面（表单）
