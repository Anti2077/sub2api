import { describe, expect, it } from 'vitest'
import { renderWikiMarkdown } from '../markdown'

describe('Wiki Markdown rendering', () => {
  it('sanitizes executable HTML and unsafe links', () => {
    const result = renderWikiMarkdown('# Title\n\n<img src=x onerror="alert(1)">\n\n[bad](javascript:alert(1))')
    expect(result.html).not.toContain('onerror')
    expect(result.html).not.toContain('javascript:')
  })

  it('generates stable unique heading IDs', () => {
    const result = renderWikiMarkdown('## 配置\n\n### 地址\n\n## 配置')
    expect(result.headings).toEqual([
      { id: '配置', text: '配置', level: 2 },
      { id: '地址', text: '地址', level: 3 },
      { id: '配置-2', text: '配置', level: 2 },
    ])
    expect(result.html).toContain('id="配置-2"')
  })

  it('opens external links without granting opener access', () => {
    const result = renderWikiMarkdown('[official](https://example.com/docs)')
    expect(result.html).toContain('target="_blank"')
    expect(result.html).toContain('rel="noopener noreferrer"')
  })
})
