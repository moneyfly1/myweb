import DOMPurify from 'dompurify'

const URI_PATTERN = /^(?:(?:https?|mailto|tel|sms|cid):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i

const BASE_TAGS = [
  'a',
  'b',
  'blockquote',
  'br',
  'code',
  'div',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'i',
  'li',
  'ol',
  'p',
  'pre',
  'span',
  'strong',
  'u',
  'ul',
]

const TABLE_TAGS = ['table', 'thead', 'tbody', 'tr', 'th', 'td']
const IMAGE_TAGS = ['img']
const EMAIL_TAGS = ['center', 'hr']

const BASE_ATTRS = ['href', 'target', 'rel']
const IMAGE_ATTRS = ['src', 'alt', 'width', 'height']
const TABLE_ATTRS = ['align', 'border', 'cellpadding', 'cellspacing']

const cache = new Map()
const MAX_CACHE_SIZE = 30

DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A') {
    const href = node.getAttribute('href')
    if (!href || !URI_PATTERN.test(href)) {
      node.removeAttribute('href')
    }
    if (node.getAttribute('target') === '_blank') {
      node.setAttribute('rel', 'noopener noreferrer')
    }
  }

  if (node.tagName === 'IMG') {
    const src = node.getAttribute('src')
    if (!src || !/^(https?:|cid:|data:image\/(?:png|gif|jpeg|jpg|webp);base64,)/i.test(src)) {
      node.removeAttribute('src')
    }
  }
})

function cacheKey(html, mode) {
  return `${mode}:${html.length}:${html.slice(0, 120)}`
}

function sanitize(html, mode, config) {
  if (!html || typeof html !== 'string') return ''
  const key = cacheKey(html, mode)
  const cached = cache.get(key)
  if (cached?.original === html) return cached.value

  try {
    const value = DOMPurify.sanitize(html, {
      ALLOWED_URI_REGEXP: URI_PATTERN,
      ALLOW_DATA_ATTR: false,
      FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form', 'input', 'button'],
      FORBID_ATTR: ['style', 'class', 'id'],
      KEEP_CONTENT: true,
      SAFE_FOR_TEMPLATES: true,
      ...config,
    })

    if (cache.size >= MAX_CACHE_SIZE) {
      cache.delete(cache.keys().next().value)
    }
    cache.set(key, { original: html, value })
    return value
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.error('sanitizeHtml failed:', error)
    }
    return ''
  }
}

export function sanitizeBasicHtml(html) {
  return sanitize(html, 'basic', {
    ALLOWED_TAGS: BASE_TAGS,
    ALLOWED_ATTR: BASE_ATTRS,
  })
}

export function sanitizeArticleHtml(html) {
  return sanitize(html, 'article', {
    ALLOWED_TAGS: [...BASE_TAGS, ...TABLE_TAGS, ...IMAGE_TAGS],
    ALLOWED_ATTR: [...BASE_ATTRS, ...TABLE_ATTRS, ...IMAGE_ATTRS],
  })
}

export function sanitizeEmailHtml(html) {
  return sanitize(html, 'email', {
    ALLOWED_TAGS: [...BASE_TAGS, ...TABLE_TAGS, ...IMAGE_TAGS, ...EMAIL_TAGS],
    ALLOWED_ATTR: [...BASE_ATTRS, ...TABLE_ATTRS, ...IMAGE_ATTRS],
  })
}

export function sanitizePlainText(text) {
  if (!text) return ''
  return DOMPurify.sanitize(String(text), { ALLOWED_TAGS: [], ALLOWED_ATTR: [] })
}
