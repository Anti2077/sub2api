interface Options {
  particleCount?: number
  spread?: number
  startVelocity?: number
  ticks?: number
  gravity?: number
  scalar?: number
  colors?: string[]
  disableForReducedMotion?: boolean
  angle?: number
  origin?: { x?: number; y?: number }
}
interface Cannon {
  (options: Options): Promise<null> | null
  reset(): void
}
declare const confetti: Cannon & {
  create(canvas: HTMLCanvasElement, options: { resize: boolean }): Cannon
}
export default confetti
