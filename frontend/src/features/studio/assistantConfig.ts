/**
 * 研究助手配置运行时借用（DR-009，design_docs/05 §6/§7）：
 * 助手槽三字段全空时，识图/求解辅助按 黑→红 优先级借用对战配置——
 * 仅存在于本次请求内存，**永不写入助手槽**（不落盘，DR-012 同构语义）；
 * authSlot 随来源槽（DR-010 掩码注入闭环）。助手槽部分填写 = 独立无效
 * 配置，不借用（沿 DR-012 边界，由既有校验/错误提示拦截）。
 */
import { isEmptyLlmConfig } from '@packages/llm'
import type { LlmEndpointConfig, SecureSlot } from '@shared/ipc/types'

/** 借用来源（null = 三槽全空，维持"请先配置"提示）。 */
export type AssistantConfigSource = 'assistant' | 'black' | 'red' | null

export interface ResolvedAssistantConfig {
  /** 实际生效的配置；null = 三槽全空（调用方维持"请先配置"提示）。 */
  config: LlmEndpointConfig | null
  /** 掩码 Key 回读时主进程注入鉴权的槽位（DR-010）：借用时为来源对战槽。 */
  authSlot: SecureSlot | undefined
  /** 生效配置来源；null = 三槽全空。 */
  source: AssistantConfigSource
}

const ASSISTANT_SLOT: SecureSlot = 'llm_config_assistant'
const BLACK_SLOT: SecureSlot = 'llm_config_black'
const RED_SLOT: SecureSlot = 'llm_config_red'

/**
 * 解析研究助手实际生效配置（DR-009）：助手槽三字段全空 → 借用
 * 黑方（非空）→ 红方（非空）→ 三槽全空返回 null；助手槽非空（含部分
 * 填写）→ 原样返回助手槽（不借用）。借用判定 = isEmptyLlmConfig 三字段
 * 全空（preset 不参与，DR-012 同口径）。
 */
export function resolveAssistantConfig(
  assistant: LlmEndpointConfig | null,
  black: LlmEndpointConfig | null,
  red: LlmEndpointConfig | null
): ResolvedAssistantConfig {
  if (assistant !== null && !isEmptyLlmConfig(assistant)) {
    return { config: assistant, authSlot: ASSISTANT_SLOT, source: 'assistant' }
  }
  if (black !== null && !isEmptyLlmConfig(black)) {
    return { config: black, authSlot: BLACK_SLOT, source: 'black' }
  }
  if (red !== null && !isEmptyLlmConfig(red)) {
    return { config: red, authSlot: RED_SLOT, source: 'red' }
  }
  return { config: null, authSlot: undefined, source: null }
}

/** 借用来源的中文显示名（提示文案用；source=assistant/null 无需显示）。 */
export function assistantSourceLabel(source: AssistantConfigSource): string {
  switch (source) {
    case 'black':
      return '黑方'
    case 'red':
      return '红方'
    default:
      return ''
  }
}
