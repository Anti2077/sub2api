import confetti from '@/third-party/canvas-confetti'

export interface CeremonyOptions {
  canvas: HTMLCanvasElement
  onProgress: (progress: number) => void
  onDone: () => void
}
/** Balance count-up followed by School Pride streamers. No code/particle transfer. */
export function startCeremony({
  canvas,
  onProgress,
  onDone,
}: CeremonyOptions): () => void {
  const fire = confetti.create(canvas, { resize: true })
  let raf = 0
  let stopped = false
  let lastBurst = 0
  const start = performance.now()
  const abort = new AbortController()
  const media = window.matchMedia('(prefers-reduced-motion: reduce)')
  const stop = () => {
    if (stopped) return
    stopped = true
    cancelAnimationFrame(raf)
    abort.abort()
    media.removeEventListener('change', stop)
    fire.reset()
    onProgress(1)
    onDone()
  }
  document.addEventListener('visibilitychange', stop, { signal: abort.signal })
  media.addEventListener('change', stop)
  const tick = (now: number) => {
    if (stopped) return
    try {
      const t = (now - start) / 1000
      onProgress(Math.min(1, t / 0.55))
      if (t > 0.55 && t < 2.3 && now - lastBurst >= 32) {
        lastBurst = now
        const options = {
          particleCount: 3,
          spread: 55,
          startVelocity: 46,
          ticks: 180,
          gravity: 0.9,
          colors: ['#ff5252', '#ff8a34', '#ffd43b', '#a3e635', '#2dd4bf', '#22d3ee', '#4385ff', '#9b6dff', '#ed64dd', '#ff80ab'],
          disableForReducedMotion: true,
        }
        fire({ ...options, angle: 60, origin: { x: 0, y: 0.65 } })
        fire({ ...options, angle: 120, origin: { x: 1, y: 0.65 } })
      }
      if (t < 3.3) raf = requestAnimationFrame(tick)
      else stop()
    } catch {
      stop()
    }
  }
  raf = requestAnimationFrame(tick)
  return stop
}
