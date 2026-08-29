import { createQRCodeDataURL } from './qrcode'

/**
 * 构造支付宝 App 唤起链接
 * 来源：Orders.vue:1009 / Packages.vue:1443 / UpgradeDevicesDrawer.vue:635 / Dashboard.vue:1141
 * @param {string} url 支付链接
 * @returns {string} alipays://platformapi/startapp?saId=10000007&qrcode=...
 */
export function buildAlipayAppUrl(url) {
  return `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(url || '')}`
}

/**
 * 判断支付链接是否需要页面跳转（而不是生成二维码）
 * 来源：Packages.vue:691-697 与 UpgradeDevicesDrawer.vue:348-354（逐字相同）
 * @param {string} url
 * @returns {boolean}
 */
export function isPaymentPageUrl(url) {
  if (!url) return false
  const value = String(url).toLowerCase()
  return value.includes('payapi/pay/payment') ||
    value.includes('submit.php') ||
    (value.startsWith('http') && !value.includes('qrcode') && !value.includes('qr.alipay') && !value.startsWith('weixin://') && !value.startsWith('wxp://'))
}

const DEFAULT_QR_OPTIONS = {
  width: 256,
  margin: 2,
  color: {
    dark: '#000000',
    light: '#FFFFFF'
  },
  errorCorrectionLevel: 'M'
}

/**
 * 生成支付二维码 DataURL（封装 createQRCodeDataURL + 默认支付二维码参数）
 * 来源：Orders.vue:861-871/:940-954、Packages.vue:1505-1515、Dashboard.vue:1224-1234、UpgradeDevicesDrawer.vue:511-517
 * @param {string} url 支付链接
 * @param {object} [options] 覆盖/追加的二维码参数（width/margin/color/errorCorrectionLevel 等）
 * @returns {Promise<string>} data:image/png;base64,...
 */
export function createPaymentQRCode(url, options = {}) {
  return createQRCodeDataURL(url, {
    ...DEFAULT_QR_OPTIONS,
    ...options,
    color: { ...DEFAULT_QR_OPTIONS.color, ...(options.color || {}) }
  })
}

/**
 * 二维码 img src 兜底：非 data: 前缀（http 链接）时附加时间戳防缓存
 * 来源：Orders.vue:365 / Packages.vue:454 / UpgradeDevicesDrawer.vue:261 模板中的三目表达式
 * @param {string} code
 * @returns {string}
 */
export function qrDisplaySrc(code) {
  if (!code) return ''
  return code.startsWith('data:') ? code : `${code}?t=${Date.now()}`
}

export default {
  buildAlipayAppUrl,
  isPaymentPageUrl,
  createPaymentQRCode,
  qrDisplaySrc
}
