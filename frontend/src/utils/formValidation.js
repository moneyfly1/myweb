/**
 * 统一的表单验证规则
 */

/**
 * 通用表单布局配置
 */
export const FORM_LAYOUT = {
  // 桌面端标签宽度
  labelWidth: '100px',
  
  // 移动端响应式配置
  mobileLabelWidth: '0px',
  
  // 表单项间距
  itemMarginBottom: '18px',
  
  // 移动端表单项间距
  mobileItemMarginBottom: '14px'
}

/**
 * 获取响应式表单布局
 * @param {boolean} isMobile - 是否为移动端
 * @returns {Object} 表单布局配置
 */
export function getFormLayout(isMobile) {
  return {
    labelWidth: isMobile ? FORM_LAYOUT.mobileLabelWidth : FORM_LAYOUT.labelWidth,
    'label-position': isMobile ? 'top' : 'right'
  }
}

/**
 * 常用验证规则
 */
export const VALIDATION_RULES = {
  // 必填
  required: (message = '此项为必填项') => [
    { required: true, message, trigger: 'blur' }
  ],
  
  // 邮箱
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ],
  
  // 手机号
  phone: [
    { required: true, message: '请输入手机号码', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入有效的手机号码', trigger: 'blur' }
  ],
  
  // 密码
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  
  // 确认密码
  confirmPassword: (passwordField = 'password') => [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        // 这个需要在组件中动态设置
        callback()
      },
      trigger: 'blur'
    }
  ],
  
  // URL
  url: [
    { required: true, message: '请输入URL地址', trigger: 'blur' },
    { type: 'url', message: '请输入有效的URL地址', trigger: 'blur' }
  ],
  
  // 数字
  number: (message = '请输入数字') => [
    { type: 'number', message, trigger: 'blur' }
  ],
  
  // 整数
  integer: (message = '请输入整数') => [
    { type: 'integer', message, trigger: 'blur' }
  ],
  
  // 正整数
  positiveInteger: (message = '请输入正整数') => [
    { type: 'integer', min: 1, message, trigger: 'blur' }
  ],
  
  // 金额
  amount: [
    { required: true, message: '请输入金额', trigger: 'blur' },
    { pattern: /^\d+(\.\d{1,2})?$/, message: '请输入有效的金额（最多两位小数）', trigger: 'blur' }
  ],
  
  // 用户名
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度为2-20个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5]+$/, message: '用户名只能包含字母、数字、下划线和中文', trigger: 'blur' }
  ],
  
  // 验证码
  verificationCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { len: 6, message: '验证码为6位数字', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '验证码格式不正确', trigger: 'blur' }
  ]
}

/**
 * 创建动态验证规则（用于确认密码等需要引用其他字段的场景）
 * @param {Object} formRef - 表单引用
 * @param {string} targetField - 目标字段名
 * @param {string} message - 错误消息
 * @returns {Array} 验证规则
 */
export function createConfirmRule(formRef, targetField, message = '两次输入不一致') {
  return [
    { required: true, message: `请再次输入${targetField}`, trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        const form = formRef?.value
        if (form && form[targetField] !== value) {
          callback(new Error(message))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}
