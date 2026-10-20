// SVG scenery and HTML hit targets share this orthographic world.
export const FARM_WORLD = { width: 960, height: 740, x: 503, y: 406 }
export function farmGeometry(count: number) {
  const cols = Math.ceil(Math.sqrt(Math.max(1, count)))
  const rows = Math.ceil(Math.max(1, count) / cols)
  const step = 602 / (cols + rows)
  const project = (u: number, v: number) => [FARM_WORLD.x + u - v, FARM_WORLD.y + (u + v) / 2]
  const u = cols * step / 2 + 24, v = rows * step / 2 + 24
  const corners = [project(-u, -v), project(u, -v), project(u, v), project(-u, v)]
  const road = corners.map(([x,y],i) => `${i ? 'L' : 'M'}${x} ${y}`).join(' ') + 'Z'
  const [bottomX, bottomY] = corners[2]
  const entrance = `M${bottomX} ${bottomY}l-158 79-123 62`
  const fence = Array.from({ length: 10 }, (_, i) => {
    const t = .18 + i * .82 / 9
    return [corners[0][0] + (corners[1][0] - corners[0][0]) * t, corners[0][1] + (corners[1][1] - corners[0][1]) * t - 42]
  })
  const rails = [0, 12].map(offset => `M${fence[0][0]} ${fence[0][1] + offset}L${fence[9][0]} ${fence[9][1] + offset}`).join(' ')
  const position = (index: number) => {
    const col = index % cols, row = Math.floor(index / cols)
    const [x,y] = project((col - (cols - 1) / 2) * step, (row - (rows - 1) / 2) * step)
    return { left: `${x / FARM_WORLD.width * 100}%`, top: `${y / FARM_WORLD.height * 100}%`, width: `${step * 1.91 / FARM_WORLD.width * 100}%`, zIndex: col + row + 1 }
  }
  return { road, entrance, fence, rails, position }
}
