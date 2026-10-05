/**
 * 研究助手配置运行时借用测试（DR-009，05 §6/§7）：
 * 优先级 黑→红 / 三槽全空 → null / 助手槽部分填写不借用（DR-012 边界）/ authSlot 随来源槽。
 */
import { describe, expect, it } from 'vitest'
import { assistantSourceLabel, resolveAssistantConfig } from '@renderer/features/studio/assistantConfig'
import type { LlmEndpointConfig } from '@shared/ipc/types'

const cfg = (model: string): LlmEndpointConfig => ({ baseUrl: 'https://api.example.com/v1', apiKey: '****abcd', model, preset: '' })
const EMPTY: LlmEndpointConfig = { baseUrl: '', apiKey: '', model: '', preset: '' }

describe('resolveAssistantConfig（DR-009 借用优先级 黑→红）', () => {
  it('助手槽全空 + 黑方有配置 → 借用黑方（authSlot 随黑槽）', () => {
    const r = resolveAssistantConfig(EMPTY, cfg('glm-4-flash'), null)
    expect(r.source).toBe('black')
    expect(r.config?.model).toBe('glm-4-flash')
    expect(r.authSlot).toBe('llm_config_black')
  })

  it('助手槽全空 + 黑空红有 → 借用红方', () => {
    const r = resolveAssistantConfig(EMPTY, EMPTY, cfg('deepseek-chat'))
    expect(r.source).toBe('red')
    expect(r.config?.model).toBe('deepseek-chat')
    expect(r.authSlot).toBe('llm_config_red')
  })

  it('黑红都有 → 黑方优先', () => {
    const r = resolveAssistantConfig(null, cfg('black-model'), cfg('red-model'))
    expect(r.source).toBe('black')
    expect(r.config?.model).toBe('black-model')
  })

  it('三槽全空 → null（维持"请先配置"提示，不产生 authSlot）', () => {
    const r = resolveAssistantConfig(EMPTY, EMPTY, EMPTY)
    expect(r.source).toBeNull()
    expect(r.config).toBeNull()
    expect(r.authSlot).toBeUndefined()
  })

  it('三槽全 null（secure.get 未配置语义）→ null', () => {
    const r = resolveAssistantConfig(null, null, null)
    expect(r.source).toBeNull()
    expect(r.config).toBeNull()
  })

  it('助手槽部分填写（如只有 model）→ 独立配置不借用（DR-012 边界）', () => {
    const partial: LlmEndpointConfig = { baseUrl: '', apiKey: '', model: 'glm-4-flash', preset: '' }
    const r = resolveAssistantConfig(partial, cfg('black-model'), cfg('red-model'))
    expect(r.source).toBe('assistant')
    expect(r.config).toBe(partial)
    expect(r.authSlot).toBe('llm_config_assistant')
  })

  it('助手槽完整 → 原样返回自身（优先于任何对战配置）', () => {
    const assistant = cfg('qwen-vl-max')
    const r = resolveAssistantConfig(assistant, cfg('black-model'), cfg('red-model'))
    expect(r.source).toBe('assistant')
    expect(r.config).toBe(assistant)
    expect(r.authSlot).toBe('llm_config_assistant')
  })
})

describe('assistantSourceLabel', () => {
  it('黑/红显示名；assistant/null 空串', () => {
    expect(assistantSourceLabel('black')).toBe('黑方')
    expect(assistantSourceLabel('red')).toBe('红方')
    expect(assistantSourceLabel('assistant')).toBe('')
    expect(assistantSourceLabel(null)).toBe('')
  })
})
