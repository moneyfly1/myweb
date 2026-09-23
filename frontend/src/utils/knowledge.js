/**
 * 知识库内容读取工具
 *
 * 定位：知识库是全站「可阅读内容」的唯一来源，帮助中心与软件教程
 * 都从这里取正文，避免同一篇教程在多处硬编码维护。
 *
 * 教程绑定方式：客户端注册表用 tutorialTitle（文章标题）声明绑定，
 * 这里把「客户端教程」分类下的文章标题建索引，运行时按标题解析成文章 id。
 * 好处：不依赖固定的文章 id（换环境/重建数据库都不会断），
 * 后台改名后也能通过模糊匹配继续找到。
 */
import { knowledgeAPI } from '@/utils/api'
import { unwrapList } from '@/utils/format'
import { apiCache } from '@/utils/apiCache'

// 客户端教程所在分类名（后台可改，改这里即可）
const CLIENT_TUTORIAL_CATEGORY = '客户端教程'

// 缓存统一走 apiCache（自带 TTL 与并发去重），不再自己维护 categoryCache /
// tutorialIndexCache / contentCache 三份无失效机制的进程内缓存：
// 之前只有 force 参数能绕过，而没有任何地方传 force，
// 管理员改完文章后用户会一直看到旧正文直到整页刷新。
// TTL 设 5 分钟：知识库改动不频繁，改完最多 5 分钟生效，无需整页刷新。
const KNOWLEDGE_TTL = 5 * 60 * 1000
const CACHE_KEYS = {
  categories: 'knowledge:categories',
  tutorialIndex: 'knowledge:tutorial-index',
  tutorialContent: title => `knowledge:tutorial:${title}`,
}
// apiCache.wrap 把 null 视为未命中，因此"文章不存在"要缓存成标记对象
const NOT_FOUND = { __knowledgeNotFound: true }

// 登录/登出等会话切换时整体失效知识库缓存
export function clearKnowledgeCache() {
  apiCache.deletePrefix('knowledge:')
}

function unwrap(response) {
  const data = response?.data
  if (!data) return null
  if (data.success === false) return null
  return data.data ?? data
}

// getCategories 知识库分类
export async function getCategories({ force = false } = {}) {
  if (force) apiCache.delete(CACHE_KEYS.categories)
  return apiCache.wrap(
    CACHE_KEYS.categories,
    async () => unwrapList(unwrap(await knowledgeAPI.getCategories())),
    KNOWLEDGE_TTL
  )
}

// findCategoryIdByName 按分类名找 id
async function findCategoryIdByName(name) {
  const categories = await getCategories()
  const hit = categories.find(c => String(c.name || '').trim() === String(name).trim())
  return hit ? hit.id : null
}

// asPlainText 把后端可能返回的「可空字符串」统一成字符串。
// 后端曾把 Go 的 sql.NullString 直接序列化，前端拿到的是
// {"String":"界面简洁的 Clash 客户端…","Valid":true} —— 页面就会整串花括号显示出来。
// 根因已在后端修（models.JSONString），这里再做一层防御：任何形状都降级成文本。
function asPlainText(value) {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'object') {
    if (typeof value.String === 'string') return value.String
    if (typeof value.string === 'string') return value.string
    if (typeof value.value === 'string') return value.value
  }
  return String(value)
}

// normalizeTitle 标题归一化（去空格、去全半角差异、统一小写）用于松散匹配
function normalizeTitle(title) {
  return String(title || '')
    .replace(/\s+/g, '')
    .replace(/[（）()【】[\]]/g, '')
    .toLowerCase()
}

// buildTutorialIndex 建立「教程标题 → 文章」索引
// 只在「客户端教程」分类下取文章，避免与其它分类同名文章冲突
async function buildTutorialIndex({ force = false } = {}) {
  if (force) apiCache.delete(CACHE_KEYS.tutorialIndex)

  return apiCache.wrap(CACHE_KEYS.tutorialIndex, async () => {
    const index = new Map()
    try {
      const categoryId = await findCategoryIdByName(CLIENT_TUTORIAL_CATEGORY)
      // 分类下文章通常不多，一次取全量（后端 page_size 上限内）
      const data = unwrap(
        await knowledgeAPI.getArticles({ category_id: categoryId || undefined, page: 1, page_size: 100 })
      )
      const items = unwrapList(data)
      items.forEach(item => {
        if (!item || !item.title) return
        index.set(normalizeTitle(item.title), item)
      })
    } catch {
      // 知识库不可用时静默降级：调用方会显示"教程待补充"而不是报错
    }
    return index
  }, KNOWLEDGE_TTL)
}

// findTutorialByTitle 按标题查找教程文章；找不到时退化为"包含客户端名"的模糊匹配
async function findTutorialByTitle(title) {
  const index = await buildTutorialIndex()
  if (!index.size || !title) return null

  const key = normalizeTitle(title)
  if (index.has(key)) return index.get(key)

  // 模糊匹配：文章标题中包含客户端名（如后台写成「Clash Verge 教程」）
  const bare = key.replace(/使用教程$|教程$/g, '')
  if (bare) {
    for (const [k, article] of index) {
      const bareK = k.replace(/使用教程$|教程$/g, '')
      if (bareK === bare || (bareK.includes(bare) && bare.length >= 3)) return article
    }
  }
  return null
}

// loadTutorialContent 按客户端注册表条目加载教程正文（缓存走 apiCache，带 TTL 与并发去重）
export async function loadTutorialContent(client, { force = false } = {}) {
  if (!client) return null
  const title = client.tutorialTitle
  if (!title) return null

  const key = CACHE_KEYS.tutorialContent(title)
  if (force) apiCache.delete(key)

  const cached = await apiCache.wrap(key, async () => {
    const article = await findTutorialByTitle(title)
    if (!article) return NOT_FOUND

    // 列表接口已带 content（后端返回完整模型），有则直接用，省一次请求
    if (article.content) {
      return { id: article.id, title: article.title, summary: asPlainText(article.summary), content: article.content }
    }

    try {
      const detail = unwrap(await knowledgeAPI.getArticle(article.id))
      return {
        id: article.id,
        title: detail?.title || article.title,
        summary: asPlainText(detail?.summary ?? article.summary),
        content: detail?.content || '',
      }
    } catch {
      return NOT_FOUND
    }
  }, KNOWLEDGE_TTL)

  return cached === NOT_FOUND ? null : cached
}

// loadArticles 取某分类下的文章（帮助中心"高频问题"用）
export async function loadArticles({ categoryName = '', keyword = '', pageSize = 6 } = {}) {
  const categoryId = categoryName ? await findCategoryIdByName(categoryName) : null
  const data = unwrap(
    await knowledgeAPI.getArticles({ category_id: categoryId || undefined, keyword: keyword || undefined, page: 1, page_size: pageSize })
  )
  const items = unwrapList(data)
  return { items, total: data?.total ?? items.length }
}
