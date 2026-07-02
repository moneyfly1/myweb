import fs from 'node:fs'
import path from 'node:path'

const rootDir = path.resolve(process.cwd(), 'src')
const riskyDeepPattern = /:{1,2}deep\((\.el-|\.list-|\.table-|\.main-|\.content-|\.page-|\.card|\.mobile-card|html|body)/
const scopedStylePattern = /<style\b([^>]*)>([\s\S]*?)<\/style>/g
const localPrefixPattern = /^(?:[.#][\w-]+|&|\w[\w-]*)(?:[\s>+~:[.#\w-]|$)/

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full, files)
    else if (entry.isFile() && full.endsWith('.vue')) files.push(full)
  }
  return files
}

function normalizeSelector(line) {
  return line.replace(/\s*\{.*$/, '').trim()
}

function hasLocalAncestor(stack) {
  return stack.some(frame => !frame.atRule && frame.local)
}

function isLocalSelector(selector) {
  if (!selector || selector.startsWith('@')) return false
  if (selector.startsWith(':deep') || selector.startsWith('::deep') || selector.startsWith(':global')) return false
  return /(\.|#|&|\w)/.test(selector)
}

function isAllowedGlobal(line) {
  return /:global\(\.user-layout\)\s+\.[a-z0-9-]+-container\b/.test(line)
}

function hasLocalPrefix(line) {
  const deepIndex = line.search(/:{1,2}deep\(|:global\(/)
  if (deepIndex <= 0) return false
  return line
    .slice(0, deepIndex)
    .split(',')
    .some(part => localPrefixPattern.test(part.trim()))
}

function auditStyle(content, file, styleStartLine) {
  const issues = []
  const stack = []
  const lines = content.split('\n')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()
    const lineNo = styleStartLine + i

    if (
      (riskyDeepPattern.test(trimmed) || trimmed.startsWith(':global(')) &&
      !hasLocalAncestor(stack) &&
      !hasLocalPrefix(trimmed)
    ) {
      if (!isAllowedGlobal(trimmed)) {
        issues.push({
          file,
          line: lineNo,
          source: trimmed,
        })
      }
    }

    const openCount = (line.match(/\{/g) || []).length
    const closeCount = (line.match(/\}/g) || []).length

    if (openCount > 0) {
      const selector = normalizeSelector(line)
      stack.push({
        atRule: selector.startsWith('@'),
        local: isLocalSelector(selector),
      })
    }

    for (let n = 0; n < closeCount && stack.length > 0; n++) {
      stack.pop()
    }
  }

  return issues
}

const issues = []
for (const file of walk(rootDir)) {
  const source = fs.readFileSync(file, 'utf8')
  let match
  while ((match = scopedStylePattern.exec(source)) !== null) {
    const attrs = match[1]
    if (!/\bscoped\b/.test(attrs)) continue
    const before = source.slice(0, match.index)
    const startLine = before.split('\n').length + 1
    issues.push(...auditStyle(match[2], path.relative(process.cwd(), file), startLine))
  }
}

if (issues.length) {
  console.error('Found scoped style selectors that can leak globally after route CSS is loaded:')
  for (const issue of issues) {
    console.error(`  ${issue.file}:${issue.line} ${issue.source}`)
  }
  console.error('Prefix root :deep/:global rules with the page/component root class.')
  process.exit(1)
}

console.log('Style scope audit passed.')
