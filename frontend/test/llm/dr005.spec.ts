/**
 * DR-005 思维链强制关闭——渲染层请求构造断言（Go 版新增用例）：
 * buildChatRequest 按预设恒发关闭参数、无任何开关路径；请求体其余字段
 * 沿 05 §3.1。Go 侧同映射断言在 internal/llm/config_test.go（双端同步锁定）。
 */
import { describe, it, expect } from 'vitest'
import { buildChatRequest, requestUrl, thinkingStyleFor } from '@packages/llm'
import type { LlmEndpointConfig } from '@shared/ipc/types'

const cfg = (preset: string): LlmEndpointConfig => ({
  baseUrl: 'https://api.example.com/v1',
  apiKey: '',
  model: 'test-model',
  preset
})

const bodyOf = (preset: string, useV2 = false): Record<string, unknown> =>
  JSON.parse(buildChatRequest(cfg(preset), 'S', 'U', { useV2 }).body) as Record<string, unknown>

describe('buildChatRequest DR-005 关闭参数恒发（按预设映射）', () => {
  it('智谱 GLM → thinking:{type:disabled}，无 enable_thinking', () => {
    const body = bodyOf('智谱 GLM')
    expect(body['thinking']).toEqual({ type: 'disabled' })
    expect('enable_thinking' in body).toBe(false)
  })

  it('智谱 GLM-4.5V（视觉）→ glm 形态', () => {
    expect(bodyOf('智谱 GLM-4.5V（视觉）')['thinking']).toEqual({ type: 'disabled' })
  })

  it('DeepSeek/Kimi/OpenAI/OpenRouter/自定义/未知 → enable_thinking:false 兜底', () => {
    for (const preset of ['DeepSeek', 'Kimi（Moonshot）', 'OpenAI', 'OpenRouter', '自定义', '']) {
      const body = bodyOf(preset)
      expect(body['enable_thinking']).toBe(false)
      expect('thinking' in body).toBe(false)
    }
  })

  it('DashScope 视觉预设（通义千问 VL）→ enable_thinking:false', () => {
    expect(bodyOf('通义千问 VL（阿里云百炼）')['enable_thinking']).toBe(false)
  })

  it('公共字段不变：temperature 0.3 / stream true / max_tokens 随 v1v2', () => {
    const v1 = bodyOf('DeepSeek')
    expect(v1['temperature']).toBe(0.3)
    expect(v1['stream']).toBe(true)
    expect(v1['max_tokens']).toBe(4096)
    const v2 = bodyOf('智谱 GLM', true)
    expect(v2['max_tokens']).toBe(8192)
    expect(v2['thinking']).toEqual({ type: 'disabled' })
  })
})

describe('thinkingStyleFor 预设映射', () => {
  it('glm / dashscope / default 三态', () => {
    expect(thinkingStyleFor('智谱 GLM')).toBe('glm')
    expect(thinkingStyleFor('智谱 GLM-4.5V（视觉）')).toBe('glm')
    expect(thinkingStyleFor('通义千问 VL（阿里云百炼）')).toBe('dashscope')
    expect(thinkingStyleFor('通义千问 3.8-Max（旗舰视觉，阿里云百炼）')).toBe('dashscope')
    expect(thinkingStyleFor('DeepSeek')).toBe('default')
    expect(thinkingStyleFor('自定义')).toBe('default')
    expect(thinkingStyleFor('未登记端点')).toBe('default')
  })
})

describe('requestUrl 与掩码 Key（回归锚定）', () => {
  it('URL 归一化与 AuthSlot 透传', () => {
    expect(requestUrl('https://a.com/v1/')).toBe('https://a.com/v1/chat/completions')
    const built = buildChatRequest({ ...cfg(''), apiKey: '****1234' }, 'S', 'U', {
      useV2: false,
      authSlot: 'llm_config_black'
    })
    expect(built.headers['Authorization']).toBeUndefined()
    expect(built.authSlot).toBe('llm_config_black')
  })
})
