import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createGameStore } from '@renderer/stores/createGameStore'
import { GameAutoSave } from '@renderer/stores/gameAutoSave'
import { restoreOrNewGame } from '@renderer/stores/gameRestore'
import { useGlobalSettings } from '@renderer/stores/globalSettings'
import { notifyLifecycle } from '@renderer/stores/lifecycleRegistry'
import { createMockApi } from '@renderer/api/mockAdapter'
import { pos } from '@packages/rules'

// T2.5 验收：保存/恢复状态机全路径（07 §2 生命周期映射 + 死局清理 game_restore.dart:32）。

const FEN_DEAD_BLACK = '4k4/3P1P3/4P4/9/9/9/9/9/9/3K5 b - - 0 1' // 黑困毙 → 红胜（死局）

async function flush(): Promise<void> {
  await new Promise((r) => setTimeout(r, 10))
}

describe('自动保存/恢复状态机', () => {
  let repo: ReturnType<typeof createMockApi>['db']

  beforeEach(() => {
    const api = createMockApi()
    repo = api.db
    useGlobalSettings.setState({ autoSave: true, loaded: true })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  function makeAutoSave(store: ReturnType<typeof createGameStore>, canSave?: () => boolean): GameAutoSave {
    return new GameAutoSave({ mode: 'humanVsHuman', vm: store.getState().vm, canSave, repo })
  }

  it('走子后 saveOnExit 写入模式桶（loadLatest 回读起始 FEN 与四元组）', async () => {
    const store = createGameStore({ mode: 'humanVsHuman' })
    store.getState().vm.playMove(pos(7, 7), pos(4, 7)) // 炮二平五
    const autoSave = makeAutoSave(store)
    autoSave.saveOnExit()
    await flush()

    const saved = await repo.loadLatest('humanVsHuman')
    expect(saved).not.toBeNull()
    // DR-008：存档 fen 为本局起始 FEN（restore 据此重放重建整局）
    expect(saved!.fen).toBe('rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1')
    expect(saved!.moves).toEqual([[7, 7, 4, 7]])
  })

  it('全局开关关闭：saveOnExit 不写；saveManual 仍写（不受开关限制）', async () => {
    useGlobalSettings.setState({ autoSave: false, loaded: true })
    const store = createGameStore({ mode: 'humanVsHuman' })
    store.getState().vm.playMove(pos(7, 7), pos(4, 7))
    const autoSave = makeAutoSave(store)

    autoSave.saveOnExit()
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).toBeNull()

    expect(autoSave.saveManual()).toBe(true)
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).not.toBeNull()
  })

  it('canSave=false（棋谱续战来源）：自动与手动保存均不写模式桶（防错 #6）', async () => {
    const store = createGameStore({ mode: 'humanVsHuman', initialFen: FEN_DEAD_BLACK })
    const autoSave = makeAutoSave(store, () => store.getState().config.initialFen === undefined)
    store.getState().vm.playMove(pos(4, 9), pos(5, 9))

    autoSave.saveOnExit()
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).toBeNull()

    expect(autoSave.saveManual()).toBe(false)
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).toBeNull()
  })

  it('生命周期事件 blur/minimize/before-quit 触发 saveOnExit；dispose 后注销', async () => {
    const store = createGameStore({ mode: 'humanVsHuman' })
    store.getState().vm.playMove(pos(7, 7), pos(4, 7))
    const autoSave = makeAutoSave(store)
    autoSave.attach()

    notifyLifecycle('blur')
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).not.toBeNull()

    // dispose：先 saveOnExit 再注销；之后生命周期事件不再写入
    await repo.deleteForMode('humanVsHuman')
    store.getState().vm.playMove(pos(7, 0), pos(6, 2))
    autoSave.dispose()
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).not.toBeNull() // dispose 自身的离开保存

    await repo.deleteForMode('humanVsHuman')
    notifyLifecycle('minimize')
    await flush()
    expect(await repo.loadLatest('humanVsHuman')).toBeNull() // 已注销，不再保存
  })

  it('restoreOrNewGame：有存档且 FEN 有效 → restored（重放重建整局）', async () => {
    const source = createGameStore({ mode: 'humanVsHuman' })
    source.getState().vm.playMove(pos(7, 7), pos(4, 7))
    source.getState().vm.playMove(pos(7, 0), pos(6, 2))
    const saver = makeAutoSave(source)
    saver.saveManual()
    await flush()

    const target = createGameStore({ mode: 'humanVsHuman' })
    const outcome = await restoreOrNewGame({ mode: 'humanVsHuman', vm: target.getState().vm, repo })
    expect(outcome).toBe('restored')
    // DR-008：以存档起始 FEN 重放重建整局——局面、历史均恢复到存档时点
    expect(target.getState().fen).toBe(source.getState().fen)
    expect(target.getState().moveHistory).toHaveLength(2)
    // 恢复后悔棋可用（跨恢复点回退到开局）
    target.getState().vm.undo()
    expect(target.getState().moveHistory).toHaveLength(1)
  })

  it('恢复存档已分胜负（死局）→ 删存档并开新局（game_restore.dart:32-37）', async () => {
    // 直接落一个死局存档：黑困毙局面 + 空历史
    await repo.saveGame({ mode: 'humanVsHuman', fen: FEN_DEAD_BLACK, moves: [] })
    await flush()

    const store = createGameStore({ mode: 'humanVsHuman' })
    const outcome = await restoreOrNewGame({ mode: 'humanVsHuman', vm: store.getState().vm, repo })

    expect(outcome).toBe('newGame')
    expect(store.getState().result).toBeNull() // 已开新局
    expect(store.getState().fen).toContain('rnbakabnr')
    expect(await repo.loadLatest('humanVsHuman')).toBeNull() // 死局存档已删除
  })

  it('存档 FEN 无效 → 开新局', async () => {
    await repo.saveGame({ mode: 'humanVsHuman', fen: '不是 FEN', moves: [] })
    await flush()
    const store = createGameStore({ mode: 'humanVsHuman' })
    const outcome = await restoreOrNewGame({ mode: 'humanVsHuman', vm: store.getState().vm, repo })
    expect(outcome).toBe('newGame')
    expect(store.getState().fen).toContain('rnbakabnr')
  })

  it('存储异常 → 开新局不崩溃（页面降级路径）', async () => {
    const brokenRepo = {
      loadLatest: async () => {
        throw new Error('sqlite not available')
      },
      deleteForMode: async () => {}
    }
    const store = createGameStore({ mode: 'humanVsHuman' })
    const outcome = await restoreOrNewGame({
      mode: 'humanVsHuman',
      vm: store.getState().vm,
      repo: brokenRepo
    })
    expect(outcome).toBe('newGame')
    expect(store.getState().fen).toContain('rnbakabnr')
  })

  it('无存档 → 开新局', async () => {
    const store = createGameStore({ mode: 'humanVsHuman' })
    const outcome = await restoreOrNewGame({ mode: 'humanVsHuman', vm: store.getState().vm, repo })
    expect(outcome).toBe('newGame')
    expect(store.getState().moveHistory).toHaveLength(0)
  })
})
