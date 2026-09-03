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
    FORBID_TAGS: ['iframe', 'object', 'embed', 'style'],
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

  const wrapper = document.createElement('div')
  wrapper.append(template.content.cloneNode(true))
  return { html: wrapper.innerHTML, headings }
}
