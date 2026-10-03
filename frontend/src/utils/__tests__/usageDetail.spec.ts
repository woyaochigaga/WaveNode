import { describe, expect, it } from 'vitest'
import { classifyUsageRequest, classifyUsageResponse } from '@/utils/usageDetail'

describe('usage detail classification', () => {
  it('classifies messages, media, tools and parameters in a request', () => {
    const parts = classifyUsageRequest(JSON.stringify({
      model: 'gpt-test',
      temperature: 0.2,
      messages: [
        { role: 'system', content: 'follow policy' },
        { role: 'user', content: [
          { type: 'input_text', text: 'describe this' },
          { type: 'input_image', image_url: '<data URI omitted>' },
        ] },
      ],
      tools: [{ type: 'function', function: { name: 'lookup' } }],
    }))

    expect(parts.some((part) => part.type === 'text' && part.role === 'system')).toBe(true)
    expect(parts.some((part) => part.type === 'image')).toBe(true)
    expect(parts.some((part) => part.type === 'tool')).toBe(true)
    expect(parts.some((part) => part.type === 'parameters' && part.content.includes('temperature'))).toBe(true)
  })

  it('merges adjacent streamed text deltas', () => {
    const body = [
      'data:{"type":"response.output_text.delta","delta":"hel"}',
      'data:{"type":"response.output_text.delta","delta":"lo"}',
      'data:[DONE]',
    ].join('\n')
    const parts = classifyUsageResponse(body)

    expect(parts).toHaveLength(1)
    expect(parts[0]).toMatchObject({ type: 'text', content: 'hello' })
  })

  it('keeps reasoning and tool calls separate from visible text', () => {
    const parts = classifyUsageResponse(JSON.stringify({
      output: [
        { type: 'reasoning', thinking: 'internal summary' },
        { type: 'function_call', arguments: '{"city":"Shanghai"}' },
        { type: 'output_text', text: 'Sunny' },
      ],
    }))

    expect(parts.map((part) => part.type)).toEqual(expect.arrayContaining(['reasoning', 'tool', 'text']))
  })
})
