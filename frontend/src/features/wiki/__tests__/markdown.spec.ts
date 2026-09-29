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

  it('keeps only local wiki images with alt text and lazy loading', () => {
    const result = renderWikiMarkdown([
      '![Network settings](/wiki/images/rustdesk/network.png)',
      '![External](https://example.com/tracker.png)',
      '![Traversal](/wiki/images/../secret.png)',
      '<img src="/wiki/images/rustdesk/config.png" alt="Configuration" srcset="https://example.com/tracker.png">',
      '<img src="/wiki/images/rustdesk/menu.png">',
    ].join('\n\n'))
    expect(result.html).toContain('src="/wiki/images/rustdesk/network.png"')
    expect(result.html).toContain('src="/wiki/images/rustdesk/config.png"')
    expect(result.html).toContain('loading="lazy"')
    expect(result.html).toContain('decoding="async"')
    expect(result.html).toContain('href="/wiki/images/rustdesk/network.png"')
    expect(result.html).toContain('aria-label="查看原图：Network settings"')
    expect(result.html).toContain('target="_blank"')
    expect(result.html).not.toContain('example.com/tracker.png')
    expect(result.html).not.toContain('secret.png')
    expect(result.html).not.toContain('menu.png')
    expect(result.html).not.toContain('srcset')
  })

  it('adds a copy control to code blocks', () => {
    const result = renderWikiMarkdown('```text\nserver-config\n```')
    expect(result.html).toContain('data-wiki-copy=""')
    expect(result.html).toContain('aria-label="复制代码"')
    expect(result.html).toContain('server-config')
  })
})
