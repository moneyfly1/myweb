import fs from 'node:fs'
import path from 'node:path'

const ROOT = path.resolve(process.cwd(), 'src')
const TARGET_DIRS = [path.join(ROOT, 'views'), path.join(ROOT, 'components')]
const VUE_EXT = '.vue'

function walk(dir, files = []) {
  if (!fs.existsSync(dir)) return files
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(fullPath, files)
      continue
    }
    if (entry.isFile() && fullPath.endsWith(VUE_EXT)) files.push(fullPath)
  }
  return files
}

function getTagName(line) {
  const m = line.match(/<\s*([a-zA-Z0-9-]+)/)
  return m ? m[1] : 'unknown'
}

function parseTemplateClicks(template) {
  const clicks = []
  const lines = template.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!line.includes('@click')) continue
    const tag = getTagName(line)
    // Match @click / @click.stop / @click.prevent and capture the value string.
    // eslint-disable-next-line security/detect-unsafe-regex
    const regex = /@click(?:\.[a-zA-Z-]+)*\s*=\s*"([^"]+)"/g
    let m
    while ((m = regex.exec(line)) !== null) {
      const expr = m[1].trim()
      clicks.push({ expr, line: i + 1, tag })
    }
  }
  return clicks
}

function parseButtonAntiPatterns(template) {
  const issues = []
  const lines = template.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!line.includes('<el-button') && !line.includes('<button')) continue
    if (line.includes(':disabled="false"') || line.includes("disabled='false'") || line.includes('disabled="false"')) {
      issues.push({
        line: i + 1,
        kind: 'redundant-disabled-false',
        source: line.trim()
      })
    }
  }
  return issues
}

function parseScriptHandlers(script) {
  const names = new Set()
  // const fn = (...) => { ... }
  // eslint-disable-next-line security/detect-unsafe-regex
  for (const m of script.matchAll(/\bconst\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>/g)) {
    names.add(m[1])
  }
  // function fn(...) { ... }
  // eslint-disable-next-line security/detect-unsafe-regex
  for (const m of script.matchAll(/\bfunction\s+([A-Za-z_$][\w$]*)\s*\(/g)) {
    names.add(m[1])
  }
  // Options API method/property functions: foo(...) { ... }
  // eslint-disable-next-line security/detect-unsafe-regex
  for (const m of script.matchAll(/\b([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*{/g)) {
    names.add(m[1])
  }
  return names
}

function normalizeExpression(expr) {
  const trimmed = expr.trim()
  // Skip clearly inline expressions; we only validate direct handler references.
  if (
    trimmed.includes('=') ||
    trimmed.includes('?') ||
    trimmed.includes('&&') ||
    trimmed.includes('||') ||
    trimmed.includes(';')
  ) {
    return null
  }
  // fn(args) -> fn
  const callMatch = trimmed.match(/^([A-Za-z_$][\w$]*)\s*\(/)
  if (callMatch) return callMatch[1]
  // bare identifier
  if (/^[A-Za-z_$][\w$]*$/.test(trimmed)) return trimmed
  return null
}

function extractSection(content, tag) {
  const re = new RegExp(`<${tag}[^>]*>([\\s\\S]*?)<\\/${tag}>`, 'm')
  const m = content.match(re)
  return m ? m[1] : ''
}

function auditFile(file) {
  const content = fs.readFileSync(file, 'utf8')
  const template = extractSection(content, 'template')
  const script = extractSection(content, 'script')
  if (!template || !script) return { file, misses: [], buttonIssues: [], total: 0 }
  const clicks = parseTemplateClicks(template)
  const buttonIssues = parseButtonAntiPatterns(template)
  const handlers = parseScriptHandlers(script)
  const misses = []

  for (const click of clicks) {
    const name = normalizeExpression(click.expr)
    if (!name) continue
    if (!handlers.has(name)) {
      misses.push({ ...click, name })
    }
  }
  return { file, misses, buttonIssues, total: clicks.length }
}

function main() {
  const files = TARGET_DIRS.flatMap(d => walk(d))
  let totalClicks = 0
  let totalMisses = 0
  let totalButtonIssues = 0

  for (const file of files) {
    const result = auditFile(file)
    totalClicks += result.total
    if (result.misses.length > 0 || result.buttonIssues.length > 0) {
      console.log(`\n${path.relative(process.cwd(), file)}`)
    }
    if (result.misses.length > 0) {
      totalMisses += result.misses.length
      for (const miss of result.misses) {
        console.log(`  L${miss.line} <${miss.tag}> @click="${miss.expr}" -> missing handler "${miss.name}"`)
      }
    }
    if (result.buttonIssues.length > 0) {
      totalButtonIssues += result.buttonIssues.length
      for (const issue of result.buttonIssues) {
        console.log(`  L${issue.line} [${issue.kind}] ${issue.source}`)
      }
    }
  }

  console.log(`\nScanned ${files.length} Vue files, ${totalClicks} click bindings.`)
  if (totalButtonIssues > 0) {
    console.error(`Found ${totalButtonIssues} button anti-patterns.`)
  }
  if (totalMisses > 0) {
    console.error(`Found ${totalMisses} unresolved click handlers.`)
    process.exitCode = 1
    return
  }
  if (totalButtonIssues > 0) {
    process.exitCode = 1
    return
  }
  console.log('No unresolved click handlers found.')
}

main()
