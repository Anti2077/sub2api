import DOMPurify from 'dompurify'
import { marked } from 'marked'

export interface WikiHeading {
  id: string
  text: string
  level: 2 | 3
}

export interface RenderedWikiMarkdown {
  html: string
  headings: WikiHeading[]
}

function headingSlug(text: string): string {
  const slug = text
    .normalize('NFKC')
    .toLocaleLowerCase()
    .replace(/[^\p{Letter}\p{Number}]+/gu, '-')
    .replace(/^-+|-+$/g, '')
  return slug || 'section'
}

export function renderWikiMarkdown(source: string): RenderedWikiMarkdown {
  const parsed = marked.parse(source, { breaks: false, gfm: true }) as string
  const sanitized = DOMPurify.sanitize(parsed, {
    FORBID_TAGS: ['iframe', 'object', 'embed', 'style', 'picture', 'source'],
    FORBID_ATTR: ['style', 'srcset'],
  }) as string
  const template = document.createElement('template')
  template.innerHTML = sanitized
  const headings: WikiHeading[] = []
  const slugCounts = new Map<string, number>()

  template.content.querySelectorAll('h2, h3').forEach((heading) => {
    const text = heading.textContent?.trim() || ''
    const base = headingSlug(text)
    const count = slugCounts.get(base) ?? 0
    slugCounts.set(base, count + 1)
    const id = count === 0 ? base : `${base}-${count + 1}`
    heading.id = id
    headings.push({
      id,
      text,
      level: heading.tagName === 'H2' ? 2 : 3,
    })
  })

  template.content.querySelectorAll<HTMLAnchorElement>('a[href]').forEach((link) => {
    const href = link.getAttribute('href') || ''
    if (/^https?:\/\//i.test(href)) {
      link.target = '_blank'
      link.rel = 'noopener noreferrer'
    }
  })

  template.content.querySelectorAll<HTMLImageElement>('img').forEach((img) => {
    const src = img.getAttribute('src') || ''
    const alt = img.getAttribute('alt')?.trim() || ''
    if (!/^\/wiki\/images\/(?:[a-z0-9_-]+\/)*[a-z0-9_-]+\.(?:png|jpe?g|webp|avif)$/i.test(src) || !alt) {
      img.remove()
      return
    }
    img.setAttribute('loading', 'lazy')
    img.setAttribute('decoding', 'async')
    if (img.closest('a')) return
    const link = document.createElement('a')
    link.href = src
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
    link.setAttribute('aria-label', `查看原图：${alt}`)
    link.dataset.wikiImageLink = ''
    img.replaceWith(link)
    link.append(img)
  })

  template.content.querySelectorAll('pre').forEach((pre) => {
    if (!pre.querySelector('code')) return
    const button = document.createElement('button')
    button.type = 'button'
    button.textContent = '复制'
    button.setAttribute('aria-label', '复制代码')
    button.setAttribute('aria-live', 'polite')
    button.dataset.wikiCopy = ''
    pre.append(button)
  })

  const wrapper = document.createElement('div')
  wrapper.append(template.content.cloneNode(true))
  return { html: wrapper.innerHTML, headings }
}
