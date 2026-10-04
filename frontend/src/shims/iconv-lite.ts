/**
 * iconv-lite 浏览器 shim（M0，dev:web/桌面 WebView 兼容层）。
 *
 * 背景：iconv-lite 依赖 node:buffer（safer-buffer 顶层读取 Buffer.prototype），
 * 在浏览器包图中会白屏崩溃（Electron 版 dev:web 同样受影响，vite 不自动 polyfill）。
 * 前端对 iconv-lite 的唯一消费是 xqfParser 的 GB18030 解码，而 WHATWG TextDecoder
 * 原生支持 gb18030/gbk（Chromium/WebKit/Firefox 全兼容），故以别名方式替换实现：
 *   vite resolve.alias: 'iconv-lite' → 本文件（仅 dev/build 浏览器包图生效）。
 * vitest（Node 环境）不经此 shim，使用真实 iconv-lite（Buffer 可用）。
 * Go 侧 internal/parsers（M5）按 Electron 版 iconv-lite 语义逐行翻译，不受本 shim 影响。
 */
let gb18030Decoder: TextDecoder | null = null

function getDecoder(): TextDecoder {
  gb18030Decoder ??= new TextDecoder('gb18030')
  return gb18030Decoder
}

/** 解码 GBK/GB18030 字节为字符串（仅支持前端实际用到的方向）。 */
export function decode(input: Uint8Array, encoding: string): string {
  const enc = encoding.toLowerCase()
  if (enc !== 'gb18030' && enc !== 'gbk') {
    throw new Error(`iconv-lite 浏览器 shim 仅支持 gb18030/gbk 解码，收到：${encoding}`)
  }
  return getDecoder().decode(input)
}

/** 前端无用例（编码仅存在于 Node 侧测试助手，走真实 iconv-lite）。 */
export function encode(_text: string, _encoding: string): Uint8Array {
  throw new Error('iconv-lite 浏览器 shim 不支持 encode')
}
