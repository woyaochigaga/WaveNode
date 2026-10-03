export type UsageDetailPartType = 'text' | 'reasoning' | 'tool' | 'image' | 'audio' | 'file' | 'parameters' | 'json'

export interface UsageDetailPart {
  type: UsageDetailPartType
  label: string
  role?: string
  content: string
}

const REQUEST_CONTENT_KEYS = new Set([
  'system', 'system_instruction', 'instructions', 'messages', 'input', 'contents',
  'tools', 'functions', 'attachments', 'files'
])

/** 解析 JSON 或 SSE；无法结构化时保留为普通文本。 */
function parsePayload(body: string): unknown[] {
  const trimmed = body.trim()
  if (!trimmed) return []
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      return [JSON.parse(trimmed)]
    } catch {
      return [trimmed]
    }
  }
  const events: unknown[] = []
  for (const line of trimmed.split('\n')) {
    if (!line.startsWith('data:')) continue
    const payload = line.slice(5).trim()
    if (!payload || payload === '[DONE]') continue
    try {
      events.push(JSON.parse(payload))
    } catch {
      events.push(payload)
    }
  }
  return events.length > 0 ? events : [trimmed]
}

function stringify(value: unknown): string {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function inferType(typeValue: unknown, key = ''): UsageDetailPartType {
  const type = `${typeValue || ''} ${key}`.toLowerCase()
  if (type.includes('reason') || type.includes('thinking')) return 'reasoning'
  if (type.includes('tool') || type.includes('function')) return 'tool'
  if (type.includes('image')) return 'image'
  if (type.includes('audio') || type.includes('voice')) return 'audio'
  if (type.includes('file') || type.includes('document')) return 'file'
  if (type.includes('text') || type.includes('message') || type.includes('content')) return 'text'
  return 'json'
}

function appendPart(parts: UsageDetailPart[], part: UsageDetailPart) {
  if (!part.content.trim()) return
  const previous = parts.at(-1)
  // 流式 delta 连续到达时合并同类片段，避免详情页出现数百个单字块。
  if (previous && previous.type === part.type && previous.label === part.label && previous.role === part.role && (part.type === 'text' || part.type === 'reasoning')) {
    previous.content += part.content
    return
  }
  parts.push(part)
}

function appendContent(parts: UsageDetailPart[], value: unknown, label: string, role?: string) {
  if (value == null) return
  if (typeof value === 'string') {
    appendPart(parts, { type: 'text', label, role, content: value })
    return
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => appendContent(parts, item, `${label} ${index + 1}`, role))
    return
  }
  if (typeof value !== 'object') {
    appendPart(parts, { type: 'text', label, role, content: String(value) })
    return
  }

  const item = value as Record<string, unknown>
  const type = inferType(item.type, label)
  const content = item.text ?? item.output_text ?? item.input_text ?? item.delta ?? item.arguments ?? item.content ?? item
  appendPart(parts, { type, label: String(item.type || label), role, content: stringify(content) })
}

/** 将不同供应商的请求体统一整理为系统指令、消息、工具、媒体和参数。 */
export function classifyUsageRequest(body: string): UsageDetailPart[] {
  const [payload] = parsePayload(body)
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return payload == null ? [] : [{ type: 'text', label: '请求内容', content: stringify(payload) }]
  }
  const value = payload as Record<string, unknown>
  const parts: UsageDetailPart[] = []

  for (const key of ['system', 'system_instruction', 'instructions']) {
    if (value[key] != null) appendContent(parts, value[key], '系统指令', 'system')
  }
  if (Array.isArray(value.messages)) {
    value.messages.forEach((message, index) => {
      const row = message as Record<string, unknown>
      appendContent(parts, row.content ?? row, `消息 ${index + 1}`, String(row.role || ''))
    })
  }
  appendContent(parts, value.input, '输入')
  appendContent(parts, value.contents, '内容')
  appendContent(parts, value.tools ?? value.functions, '工具定义')
  appendContent(parts, value.attachments ?? value.files, '附件')

  const parameters = Object.fromEntries(Object.entries(value).filter(([key]) => !REQUEST_CONTENT_KEYS.has(key)))
  if (Object.keys(parameters).length > 0) {
    appendPart(parts, { type: 'parameters', label: '请求参数', content: stringify(parameters) })
  }
  return parts
}

function collectResponseValue(parts: UsageDetailPart[], value: unknown, label = '输出') {
  if (value == null) return
  if (typeof value === 'string') {
    appendPart(parts, { type: 'text', label, content: value })
    return
  }
  if (Array.isArray(value)) {
    value.forEach((item) => collectResponseValue(parts, item, label))
    return
  }
  if (typeof value !== 'object') return

  const row = value as Record<string, unknown>
  const eventType = String(row.type || '')
  if (row.delta != null && (typeof row.delta === 'string' || eventType.includes('delta'))) {
    const delta = typeof row.delta === 'object' && row.delta !== null
      ? ((row.delta as Record<string, unknown>).text ?? (row.delta as Record<string, unknown>).content ?? row.delta)
      : row.delta
    appendPart(parts, { type: inferType(eventType, 'delta'), label: eventType || label, content: stringify(delta) })
  }
  if (row.text != null || row.output_text != null) {
    appendPart(parts, { type: inferType(eventType, 'text'), label: eventType || label, content: stringify(row.text ?? row.output_text) })
  }
  if (row.reasoning != null || row.thinking != null) {
    appendPart(parts, { type: 'reasoning', label: '推理内容', content: stringify(row.reasoning ?? row.thinking) })
  }
  if (row.tool_calls != null || row.function_call != null) {
    appendPart(parts, { type: 'tool', label: '工具调用', content: stringify(row.tool_calls ?? row.function_call) })
  }
  if ((eventType.includes('tool') || eventType.includes('function')) && (row.arguments != null || row.input != null || row.name != null)) {
    appendPart(parts, { type: 'tool', label: eventType || '工具调用', content: stringify(row.arguments ?? row.input ?? row) })
  }
  if ((eventType.includes('image') || eventType.includes('audio') || eventType.includes('file')) && row.content != null) {
    appendPart(parts, { type: inferType(eventType), label: eventType, content: stringify(row.content) })
  }

  const nestedKeys = ['message', 'content', 'output', 'choices', 'candidates', 'parts', 'response']
  nestedKeys.forEach((key) => {
    if (row[key] != null) collectResponseValue(parts, row[key], key)
  })
}

/** 合并同步 JSON 与流式 SSE 的文本、推理、工具和媒体输出。 */
export function classifyUsageResponse(body: string): UsageDetailPart[] {
  const values = parsePayload(body)
  const parts: UsageDetailPart[] = []
  values.forEach((value) => collectResponseValue(parts, value))
  if (parts.length === 0 && values.length > 0) {
    parts.push({ type: 'json', label: '响应数据', content: stringify(values.length === 1 ? values[0] : values) })
  }
  return parts
}

export function prettyUsagePayload(body: string): string {
  const trimmed = body.trim()
  if (!trimmed || !(trimmed.startsWith('{') || trimmed.startsWith('['))) return body
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return body
  }
}
