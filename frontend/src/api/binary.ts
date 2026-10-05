/**
 * Wails 通道二进制承载（Go 版 M5，06 文档适配差异）：
 *
 * Electron IPC 走结构化克隆，Uint8Array 可直传；Wails v2 的 invoke 用
 * JSON.stringify 序列化载荷，Uint8Array 会退化为 {"0":65,...} 索引对象——
 * 因此 App.CorpusReadFiles（Go→JS，[]byte → JSON base64 字符串）与
 * App.ParserParseBatch（JS→Go，Uint8Array → base64 字符串）两端约定以
 * base64 字符串承载字节，编解码收敛在本文件（仅 wails 传输路径使用，
 * mock 路径保持 Uint8Array 内存语义不变）。
 */

/** Uint8Array → base64（大数组按 0x8000 分块，规避 String.fromCharCode 展开上限）。 */
export function bytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  const chunk = 0x8000
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk))
  }
  return btoa(binary)
}

/** base64 → Uint8Array（Go []byte 的 JSON 形态解码；空串返回空数组）。 */
export function base64ToBytes(base64: string): Uint8Array {
  if (base64.length === 0) return new Uint8Array(0)
  const binary = atob(base64)
  const out = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) out[i] = binary.charCodeAt(i)
  return out
}
