/**
 * Playwright 冒烟链路（09 §2.5 / T7.3）：启动 → 主页 7 入口可见 → 双人走一着
 * （棋盘坐标点击）→ 悔棋 → 新游戏 → 棋谱库打开。
 * LLM/E2E 一律 mock（不依赖外部服务）：浏览器 mock 模式下 api 适配层探测
 * window.go 失败自动落内存 mock，链路只用本地规则内核，不触网。
 * 运行：npm run test:e2e（= playwright test；webServer 自动起 vite dev:web）。
 *
 * 与 Electron 版 e2e/smoke.spec.mjs 的差异：浏览器模式内存 mock 每次加载即干净，
 * 无 sqlite 自动存档恢复，故链首不需"新游戏清场"（链中仍保留一处覆盖该路径）。
 */
import { expect, test } from '@playwright/test'

/** 棋盘网格交点 (col,row) → svg 内坐标（boardLayout.ts computeBoardLayout 等价复算：
 *  cell = min(w/9.6, h/10.6)，交点 = 原点 + col/row × cell）。 */
function boardPoint(rect: { width: number; height: number }, col: number, row: number) {
  const cell = Math.min(rect.width / 9.6, rect.height / 10.6)
  const originX = (rect.width - 8 * cell) / 2
  const originY = (rect.height - 9 * cell) / 2
  return { x: originX + col * cell, y: originY + row * cell }
}

/** 点击棋盘交点（先取 svg 实时盒，再按比例定位） */
async function clickBoardPoint(page: import('@playwright/test').Page, col: number, row: number) {
  const rect = await page.$eval('[data-testid=board-svg]', (el) => {
    const r = el.getBoundingClientRect()
    return { width: r.width, height: r.height }
  })
  await page.click('[data-testid=board-svg]', { position: boardPoint(rect, col, row) })
}

const turnText = (page: import('@playwright/test').Page) =>
  page.$eval('.cc-turn-label-red, .cc-turn-label-black', (el) => el.textContent?.trim())

test('冒烟链路：主页 → 双人走一着 → 悔棋 → 新游戏 → 棋谱库', async ({ page }) => {
  // ---- ① 主页 7 入口可见 ----
  await page.goto('/')
  await page.waitForSelector('.cc-home-btn')
  const entries = await page.$$eval('button.cc-home-btn', (btns) =>
    btns.map((b) => b.textContent?.trim())
  )
  expect(entries).toEqual([
    '残局选关',
    '人机对战',
    '人机对战（大模型）',
    '大模型对战',
    '双人对弈',
    '残局工作室（摆盘/导入/求解）',
    '棋谱库'
  ])

  // ---- ② 进入双人对弈，棋盘就绪（浏览器 mock 无持久存档，开局即红先 0 步）----
  await page.click('button.cc-home-btn:has-text("双人对弈")')
  await page.waitForSelector('[data-testid=board-svg]')
  await page.waitForFunction(
    () =>
      document.querySelector('.cc-turn-label-red') !== null &&
      (document.querySelector('.cc-game-info')?.textContent ?? '').includes('步数: 0'),
    undefined,
    { timeout: 5000 }
  )
  expect(await turnText(page)).toBe('红方')

  // ---- ③ 双人走一着：炮二平五（h7-e7，点击选中 + 点击目标）----
  await clickBoardPoint(page, 7, 7)
  await clickBoardPoint(page, 4, 7)
  // 220ms 走子动画结束后落子，轮黑方、步数 1。
  await page.waitForFunction(
    () =>
      document.querySelector('.cc-turn-label-black') !== null &&
      (document.querySelector('.cc-game-info')?.textContent ?? '').includes('步数: 1'),
    undefined,
    { timeout: 5000 }
  )
  expect(await turnText(page)).toBe('黑方')

  // ---- ④ 悔棋：回到初始局面，红先、步数 0 ----
  await page.click('button:has-text("悔棋")')
  await page.waitForFunction(
    () =>
      document.querySelector('.cc-turn-label-red') !== null &&
      (document.querySelector('.cc-game-info')?.textContent ?? '').includes('步数: 0'),
    undefined,
    { timeout: 5000 }
  )
  expect(await turnText(page)).toBe('红方')

  // ---- ⑤ 走回一着 → 新游戏（确认对话框）→ 清空 ----
  await clickBoardPoint(page, 7, 7)
  await clickBoardPoint(page, 4, 7)
  await page.waitForSelector('.cc-turn-label-black', { timeout: 5000 })
  await page.click('button:has-text("新游戏")')
  await page.waitForSelector('text=开始新游戏', { timeout: 5000 })
  await page.click('button:has-text("确定")')
  await page.waitForFunction(
    () => (document.querySelector('.cc-game-info')?.textContent ?? '').includes('步数: 0'),
    undefined,
    { timeout: 5000 }
  )
  expect(await turnText(page)).toBe('红方')

  // ---- ⑥ 返回主页 → 棋谱库打开 ----
  await page.click('button[aria-label="返回"]')
  await page.waitForSelector('.cc-home-btn', { timeout: 5000 })
  await page.click('button.cc-home-btn:has-text("棋谱库")')
  await page.waitForSelector('h2:has-text("棋谱库")', { timeout: 5000 })
  await expect(page.locator('button[aria-label="返回"]')).toBeVisible()
})
