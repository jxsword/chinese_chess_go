/**
 * LLM 配置卡（对应 llm_config_editor.dart，08 文档 §3.3）：
 * 预设下拉 / 端点地址 / Key（掩码回显）/ 模型 ID / 测试连接。
 * 【DR-005】无思维链开关 UI——关闭参数在请求构造层按预设恒发。
 * 配置整体经 cc:secure 槽位持久化（掩码回读，DR-010 两态鉴权）。
 */
import { useState } from 'react'
import { LLM_PRESETS, LLM_PRESET_CUSTOM, type LlmPreset } from '@packages/llm'
import { api } from '@renderer/api/client'
import type { LlmEndpointConfig, SecureSlot } from '@shared/ipc/types'

export interface LlmConfigCardProps {
  /** 卡片标题（如"黑方模型（对手）"/"红方模型"） */
  title: string
  /** 持久化槽位（测试连接/掩码 Key 的主进程注入依据，DR-010） */
  slot: SecureSlot
  config: LlmEndpointConfig
  /** 配置变化（不落盘；落盘由页面防抖保存） */
  onChange: (config: LlmEndpointConfig) => void
  /** 预设清单（缺省对话模型预设；研究助手/识图传 VISION_LLM_PRESETS） */
  presets?: readonly LlmPreset[]
  /**
   * 测试连接的生效配置（DR-014 镜像）：本卡配置全空、对局时跟随另一方时，
   * 传另一方配置与槽位——"测试连接"测的是真正会用于对局的配置。
   */
  testOverride?: { config: LlmEndpointConfig; slot: SecureSlot }
}

/** 当前预设：优先按配置记录的预设名，回退按端点地址匹配（旧存档兼容）。 */
function presetFor(config: LlmEndpointConfig, presets: readonly LlmPreset[]): LlmPreset {
  const byName = presets.find((p) => p.name === config.preset)
  if (byName !== undefined) return byName
  const hit = presets.find(
    (p) => p !== LLM_PRESET_CUSTOM && p.baseUrl === config.baseUrl.trim()
  )
  return hit ?? LLM_PRESET_CUSTOM
}

export function LlmConfigCard({ title, slot, config, onChange, presets = LLM_PRESETS, testOverride }: LlmConfigCardProps): React.JSX.Element {
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<string | null>(null)

  const runTestConnection = (): void => {
    setTesting(true)
    setTestResult(null)
    void api.llm
      .testConnection(testOverride !== undefined ? testOverride.config : config, testOverride !== undefined ? testOverride.slot : slot)
      .then((res) => setTestResult(res.message))
      .catch(() => setTestResult('连接失败：测试通道不可用'))
      .finally(() => setTesting(false))
  }

  return (
    <div className="cc-card cc-llm-config" data-testid="llm-config-card">
      <div className="cc-section-title">{title}</div>
      <label className="cc-llm-field">
        <span>端点预设</span>
        <select
          aria-label={`${title}端点预设`}
          value={presetFor(config, presets).name}
          onChange={(e) => {
            const preset = presets.find((p) => p.name === e.target.value)
            if (preset === undefined) return
            // 自定义也记录预设名（兜底关闭参数形态）；非自定义回填端点与示例模型。
            onChange({
              ...config,
              preset: preset.name,
              ...(preset === LLM_PRESET_CUSTOM ? {} : { baseUrl: preset.baseUrl, model: preset.exampleModel })
            })
          }}
        >
          {presets.map((p) => (
            <option key={p.name} value={p.name}>
              {p.name}
            </option>
          ))}
        </select>
      </label>
      <label className="cc-llm-field">
        <span>端点地址（Base URL）</span>
        <input
          type="text"
          placeholder="https://…/v4 或 https://…/v1"
          value={config.baseUrl}
          onChange={(e) => onChange({ ...config, baseUrl: e.target.value })}
        />
      </label>
      <label className="cc-llm-field">
        <span>API Key</span>
        <input
          type="password"
          placeholder="留空表示本地网关；已保存的 Key 以掩码回显"
          value={config.apiKey}
          onChange={(e) => onChange({ ...config, apiKey: e.target.value })}
        />
      </label>
      <label className="cc-llm-field">
        <span>模型 ID</span>
        <input
          type="text"
          placeholder="如 glm-4-flash / deepseek-chat"
          value={config.model}
          onChange={(e) => onChange({ ...config, model: e.target.value })}
        />
      </label>
      <div className="cc-settings-hint">
        思维链已强制关闭（按端点预设发送关闭参数，无需配置）。
      </div>
      <div className="cc-button-row">
        <button
          type="button"
          className="cc-btn"
          disabled={testing}
          onClick={runTestConnection}
          data-testid="llm-test-connection"
        >
          {testing ? '测试中…' : '测试连接'}
        </button>
      </div>
      {testResult !== null && (
        <div
          className={testResult.startsWith('连接成功') ? 'cc-llm-test-ok' : 'cc-llm-test-fail'}
          data-testid="llm-test-result"
          role="status"
        >
          {testResult}
        </div>
      )}
    </div>
  )
}
