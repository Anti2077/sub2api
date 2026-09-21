import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RedeemCeremony from '../RedeemCeremony.vue'
import type { CeremonyOptions } from '../ceremony'
const { start } = vi.hoisted(() => ({ start: vi.fn() }))
vi.mock('../ceremony', () => ({ startCeremony: start }))

describe('RedeemCeremony lifecycle', () => {
  let options: CeremonyOptions
  let cancel: ReturnType<typeof vi.fn>
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: false })),
    )
    vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      x: 10,
      y: 100,
      top: 100,
      left: 10,
      width: 200,
      height: 40,
      bottom: 140,
      right: 210,
      toJSON: () => ({}),
    })
    cancel = vi.fn(() => {
      options.onProgress(1)
      options.onDone()
    })
    start.mockImplementation((input: CeremonyOptions) => {
      options = input
      return cancel
    })
  })
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('delivers the final state immediately for reduced motion without starting graphics', async () => {
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: true })),
    )
    const wrapper = mount(RedeemCeremony)
    await wrapper.vm.play()
    expect(start).not.toHaveBeenCalled()
    expect(wrapper.emitted('progress')).toEqual([[1]])
    wrapper.unmount()
  })

  it('finishes a pending play when the route unmounts and removes the overlay', async () => {
    const wrapper = mount(RedeemCeremony)
    const finished = vi.fn()
    const playing = wrapper.vm.play().then(finished)
    await flushPromises()
    expect(start).toHaveBeenCalledOnce()
    expect(document.querySelector('.celebration-canvas')).not.toBeNull()
    wrapper.unmount()
    await playing
    expect(cancel).toHaveBeenCalledOnce()
    expect(finished).toHaveBeenCalledOnce()
    expect(document.querySelector('.celebration-canvas')).toBeNull()
  })

  it('degrades renderer startup failure to a completed delivery', async () => {
    start.mockImplementation(() => {
      throw new Error('Canvas unavailable')
    })
    const wrapper = mount(RedeemCeremony)
    await wrapper.vm.play()
    expect(wrapper.emitted('progress')).toEqual([[1]])
    expect(document.querySelector('.celebration-canvas')).toBeNull()
    wrapper.unmount()
  })
})
