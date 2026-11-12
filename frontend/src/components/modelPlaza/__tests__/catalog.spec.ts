import { describe, expect, it } from 'vitest'
import { buildCatalog, displayPrice, priceValues } from '../catalog'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'

function model(name: string, input = 3e-6): PlazaModel {
  return {
    name,
    platform: 'openai',
    pricing: {
      billing_mode: 'token', input_price: input, output_price: 12e-6,
      cache_write_price: null, cache_read_price: 1e-6,
      image_input_price: null, image_output_price: null, per_request_price: null, intervals: []
    },
    official_pricing: null
  }
}

function group(id: number, rate: number, models: PlazaModel[]): ModelPlazaGroup {
  return {
    id, name: `Group ${id}`, description: '', platform: 'openai', subscription_type: 'standard',
    rate_multiplier: rate, peak_rate_enabled: false, peak_start: '', peak_end: '', peak_rate_multiplier: 1,
    is_exclusive: false, image_rate_independent: false, image_rate_multiplier: 1,
    long_context_pricing_enabled: true, models
  }
}

describe('model-first catalog', () => {
  it('merges offerings and selects the cheapest effective base price', () => {
    const catalog = buildCatalog([group(1, 1, [model('gpt')]), group(2, 0.5, [model('gpt')])])
    expect(catalog).toHaveLength(1)
    expect(catalog[0]!.offerings).toHaveLength(2)
    expect(catalog[0]!.best.group.id).toBe(2)
    expect(displayPrice(priceValues(catalog[0]!.best).primary, 'token', 1_000_000)).toBe('$1.50')
  })

  it('uses the first resolved tier, personal rate, and independent image rate', () => {
    const tiered = model('tiered')
    tiered.pricing!.intervals = [{ min_tokens: 0, max_tokens: 200000, input_price: null, output_price: null,
      cache_write_price: null, cache_read_price: null, input_multiplier: 2, per_request_price: null }]
    const personal = group(1, 1, [tiered])
    personal.user_rate_multiplier = 0.5
    expect(priceValues({ group: personal, model: tiered }).primary).toBe(3e-6)

    const image = model('image')
    image.pricing!.billing_mode = 'image'
    image.pricing!.per_request_price = 0.2
    personal.image_rate_independent = true
    personal.image_rate_multiplier = 0.25
    expect(priceValues({ group: personal, model: image }).primary).toBe(0.05)
    expect(displayPrice(0.05, 'image', 1_000)).toBe('$0.05')
  })
})
