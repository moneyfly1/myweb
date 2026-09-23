// 静态审计：前端调用的「API 方法」是否真的存在。
//
// 为什么需要（2026-09-23 线上事故）：订阅域名池面板调的是 adminAPI.getDomainPool，
// 而该方法被误加到了 settingsAPI —— 构建、lint、类型检查全都发现不了（两个都是普通
// 对象字面量），直到管理员打开「系统设置」才报
// "l.getDomainPool is not a function"（压缩后变量名），功能整块点不动。
//
// 做法：以 utils/api.js 里导出的 *API 对象为唯一真相源，收集各对象的方法名，
// 再扫描所有 src/**/*.{vue,js} 的 `xxxAPI.method(` 调用点，逐个核对是否存在。
//
// 用法：npm run audit:api  （返回码 1 表示有悬空调用）
import fs from 'node:fs'
import path from 'node:path'

const srcDir = path.resolve(process.cwd(), 'src')
const apiFile = path.join(srcDir, 'utils', 'api.js')

if (!fs.existsSync(apiFile)) {
  console.error(`找不到 ${apiFile}`)
  process.exit(1)
}

// ---- 1) 收集 utils/api.js 里每个 *API 对象的方法名 ----
const apiSource = fs.readFileSync(apiFile, 'utf8')
const exportedApiObjects = new Map() // 对象名 → Set(方法名)

const exportPattern = /export\s+const\s+(\w*API)\s*=\s*\{/g
let m
while ((m = exportPattern.exec(apiSource)) !== null) {
  const name = m[1]
  const bodyStart = m.index + m[0].length
  let depth = 1
  let i = bodyStart
  while (i < apiSource.length && depth > 0) {
    const ch = apiSource[i]
    if (ch === '{') depth++
    else if (ch === '}') depth--
    i++
  }
  const body = apiSource.slice(bodyStart, i - 1)
  const methods = new Set()
  // 只认「顶层键:」，避免把嵌套对象/箭头函数体里的名字算进来
  let nest = 0
  for (const line of body.split('\n')) {
    const trimmed = line.trim()
    if (nest === 0) {
      const km = /^([A-Za-z_$][\w$]*)\s*:/.exec(trimmed)
      if (km) methods.add(km[1])
    }
    for (const ch of line) {
      if (ch === '{' || ch === '(' || ch === '[') nest++
      else if (ch === '}' || ch === ')' || ch === ']') nest--
    }
  }
  exportedApiObjects.set(name, methods)
}

if (exportedApiObjects.size === 0) {
  console.error('未从 utils/api.js 解析出任何 *API 对象，请检查解析规则')
  process.exit(1)
}

// ---- 2) 扫描调用点 ----
function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue
      walk(full, files)
    } else if (entry.isFile() && (full.endsWith('.vue') || full.endsWith('.js'))) {
      files.push(full)
    }
  }
  return files
}

const problems = []
const names = [...exportedApiObjects.keys()]
// 同时匹配换行调用：xxxAPI\n  .method(
const callPattern = new RegExp(
  `\\b(${names.join('|')})\\s*\\.\\s*([A-Za-z_$][\\w$]*)\\s*\\(`,
  'g',
)

for (const file of walk(srcDir)) {
  if (file === apiFile) continue // 定义处自身不算调用
  const text = fs.readFileSync(file, 'utf8')
  const lines = text.split('\n')
  let hit
  callPattern.lastIndex = 0
  while ((hit = callPattern.exec(text)) !== null) {
    const [objName, method] = [hit[1], hit[2]]
    const methodSet = exportedApiObjects.get(objName)
    if (!methodSet || methodSet.has(method)) continue
    const line = text.slice(0, hit.index).split('\n').length
    problems.push({
      file: path.relative(process.cwd(), file),
      line,
      message: `${objName}.${method}() 不存在（${objName} 未定义该方法）`,
      source: (lines[line - 1] || '').trim().slice(0, 120),
    })
  }
}

// ---- 3) 报告 ----
const objects = [...exportedApiObjects.entries()]
  .map(([n, s]) => `${n}(${s.size})`)
  .join('、')

if (problems.length === 0) {
  console.log(`✅ API 方法调用审计通过：${objects}`)
  process.exit(0)
}

console.error(`❌ 发现 ${problems.length} 处悬空 API 方法调用：\n`)
for (const p of problems) {
  console.error(`  ${p.file}:${p.line}  ${p.message}`)
  if (p.source) console.error(`      ${p.source}`)
}
console.error(`\n可用对象：${objects}`)
console.error('修复：把方法加到对应的 *API 对象，或改调正确的对象。')
process.exit(1)
