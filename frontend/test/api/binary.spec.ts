// @vitest-environment jsdom
/** Wails 通道 base64 承载编解码（api/binary.ts，06 文档适配差异） */
import { describe, it, expect } from 'vitest'
import { bytesToBase64, base64ToBytes } from '@renderer/api/binary'

describe('binary base64 承载', () => {
  it('往返：任意字节', () => {
    const bytes = new Uint8Array([0, 1, 2, 250, 251, 255, 0x58, 0x51])
    expect(base64ToBytes(bytesToBase64(bytes))).toEqual(bytes)
  })

  it('空数组与空串', () => {
    expect(bytesToBase64(new Uint8Array(0))).toBe('')
    expect(base64ToBytes('')).toEqual(new Uint8Array(0))
  })

  it('跨 0x8000 分块边界（String.fromCharCode 展开上限）', () => {
    const bytes = new Uint8Array(0x8000 + 100)
    for (let i = 0; i < bytes.length; i++) bytes[i] = i % 251
    expect(base64ToBytes(bytesToBase64(bytes))).toEqual(bytes)
  })

  it('与 Go []byte 的 JSON base64 形态互通', () => {
    // Go: json.Marshal([]byte{104,105}) == "aGk="
    expect(base64ToBytes('aGk=')).toEqual(new Uint8Array([104, 105]))
    expect(bytesToBase64(new Uint8Array([104, 105]))).toBe('aGk=')
  })
})
