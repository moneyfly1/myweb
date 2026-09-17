/**
 * 用户名规则（与后端 utils.ValidateUsername / handlers.checkUsernameAvailable 对齐）
 *
 * 后端规则：^[a-zA-Z0-9_\u4e00-\u9fa5]{2,20}$
 * 后端唯一性：忽略大小写（a1ex 与 A1EX 视为同一个用户名，不允许重复注册）
 *
 * 之前各表单各写一套：注册页只允许字母数字下划线（中文用户名被前端拦下，
 * 但后端允许），个人设置页甚至没有字符校验，与实际服务端规则不一致。
 */

/** 合法用户名：2-20 位，中文、字母、数字、下划线 */
export const USERNAME_PATTERN = /^[a-zA-Z0-9_\u4e00-\u9fa5]{2,20}$/

export const USERNAME_MIN = 2
export const USERNAME_MAX = 20

export const USERNAME_HINT = '2-20 位，支持中文、字母、数字、下划线；用户名不区分大小写，不能与其他用户重复'

/** 校验单个用户名，返回错误文案；通过则返回空字符串 */
export function validateUsername(value) {
  const name = (value ?? '').trim()
  if (!name) return '请输入用户名'
  if (name.length < USERNAME_MIN || name.length > USERNAME_MAX) {
    return `用户名长度必须在 ${USERNAME_MIN} 到 ${USERNAME_MAX} 个字符之间`
  }
  if (!USERNAME_PATTERN.test(name)) {
    return '用户名只能包含中文、字母、数字和下划线'
  }
  return ''
}

/** Element Plus 表单用的校验器 */
export function usernameValidator(_rule, value, callback) {
  const message = validateUsername(value)
  if (message) {
    callback(new Error(message))
    return
  }
  callback()
}

/** Element Plus 表单规则（可直接放入 rules.username） */
export const usernameFormRules = [{ validator: usernameValidator, trigger: 'blur' }]
