const getQRCodeModule = async () => {
  const module = await import('qrcode')
  const qr = module?.default && typeof module.default.toDataURL === 'function'
    ? module.default
    : module

  if (!qr || typeof qr.toDataURL !== 'function') {
    throw new Error('二维码组件加载失败')
  }

  return qr
}

export const createQRCodeDataURL = async (value, options = {}) => {
  const text = String(value || '').trim()
  if (!text) {
    throw new Error('支付链接为空')
  }

  const qr = await getQRCodeModule()
  return qr.toDataURL(text, options)
}

export const drawQRCodeToCanvas = async (canvas, value, options = {}) => {
  const text = String(value || '').trim()
  if (!canvas || !text) return

  const qr = await getQRCodeModule()
  return qr.toCanvas(canvas, text, options)
}
