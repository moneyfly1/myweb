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

// 客户端教程所在分类名（后台可改，改这里即可）
export const CLIENT_TUTORIAL_CATEGORY = '客户端教程'

let categoryCache = null
let tutorialIndexCache = null
let tutorialIndexPromise = null

function unwrap(response) {
  const data = response?.data
  if (!data) return null
  if (data.success === false) return null
  return data.data ?? data
}

// getCategories 知识库分类（带进程内缓存）
export async function getCategories({ force = false } = {}) {
  if (categoryCache && !force) return categoryCache
  const data = unwrap(await knowledgeAPI.getCategories())
  categoryCache = Array.isArray(data) ? data : data?.categories || data?.items || []
  return categoryCache
}

// findCategoryIdByName 按分类名找 id
export async function findCategoryIdByName(name) {
  const categories = await getCategories()
  const hit = categories.find(c => String(c.name || '').trim() === String(name).trim())
  return hit ? hit.id : null
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
export async function buildTutorialIndex({ force = false } = {}) {
  if (tutorialIndexCache && !force) return tutorialIndexCache
  if (tutorialIndexPromise && !force) return tutorialIndexPromise

  tutorialIndexPromise = (async () => {
    const index = new Map()
    try {
      const categoryId = await findCategoryIdByName(CLIENT_TUTORIAL_CATEGORY)
      // 分类下文章通常不多，一次取全量（后端 page_size 上限内）
      const data = unwrap(
        await knowledgeAPI.getArticles({ category_id: categoryId || undefined, page: 1, page_size: 100 })
      )
      const items = data?.items || (Array.isArray(data) ? data : [])
      items.forEach(item => {
        if (!item || !item.title) return
        index.set(normalizeTitle(item.title), item)
      })
    } catch {
      // 知识库不可用时静默降级：调用方会显示"教程待补充"而不是报错
    }
    tutorialIndexCache = index
    tutorialIndexPromise = null
    return index
  })()

  return tutorialIndexPromise
}

// findTutorialByTitle 按标题查找教程文章；找不到时退化为"包含客户端名"的模糊匹配
export async function findTutorialByTitle(title) {
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

// loadTutorialContent 按客户端注册表条目加载教程正文（含缓存）
const contentCache = new Map()

export async function loadTutorialContent(client, { force = false } = {}) {
  if (!client) return null
  const title = client.tutorialTitle
  if (!title) return null
  if (contentCache.has(title) && !force) return contentCache.get(title)

  const article = await findTutorialByTitle(title)
  if (!article) {
    contentCache.set(title, null)
    return null
  }

  // 列表接口已带 content（后端返回完整模型），有则直接用，省一次请求
  if (article.content) {
    const result = { id: article.id, title: article.title, summary: article.summary || '', content: article.content }
    contentCache.set(title, result)
    return result
  }

  try {
    const detail = unwrap(await knowledgeAPI.getArticle(article.id))
    const result = {
      id: article.id,
      title: detail?.title || article.title,
      summary: detail?.summary || article.summary || '',
      content: detail?.content || '',
    }
    contentCache.set(title, result)
    return result
  } catch {
    return null
  }
}

// loadArticles 取某分类下的文章（帮助中心"高频问题"用）
export async function loadArticles({ categoryName = '', keyword = '', pageSize = 6 } = {}) {
  const categoryId = categoryName ? await findCategoryIdByName(categoryName) : null
  const data = unwrap(
    await knowledgeAPI.getArticles({ category_id: categoryId || undefined, keyword: keyword || undefined, page: 1, page_size: pageSize })
  )
  const items = data?.items || (Array.isArray(data) ? data : [])
  return { items, total: data?.total ?? items.length }
}

export function clearKnowledgeCache() {
  categoryCache = null
  tutorialIndexCache = null
  tutorialIndexPromise = null
  contentCache.clear()
}
