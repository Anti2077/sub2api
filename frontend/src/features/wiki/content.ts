import ccSwitchSource from '@/content/wiki/zh/clients/cc-switch.md?raw'
import codexImage2Source from '@/content/wiki/zh/clients/codex-image2.md?raw'
import firstRequestSource from '@/content/wiki/zh/getting-started/first-request.md?raw'
import openMinisSource from '@/content/wiki/zh/clients/open-minis.md?raw'
import requestErrorsSource from '@/content/wiki/zh/troubleshooting/request-errors.md?raw'

export type WikiSectionId = 'getting-started' | 'clients' | 'troubleshooting'
export type WikiArticleStatus = 'verified' | 'draft'

export interface WikiSection {
  id: WikiSectionId
  title: string
  description: string
  order: number
}

export interface WikiArticle {
  slug: string
  section: WikiSectionId
  title: string
  summary: string
  order: number
  tags: string[]
  status: WikiArticleStatus
  lastVerified: string | null
  verifiedWith?: string
  source: string
}

export const wikiSections: readonly WikiSection[] = [
  {
    id: 'getting-started',
    title: '快速开始',
    description: '先验证接口、密钥和模型，再连接日常使用的客户端。',
    order: 10,
  },
  {
    id: 'clients',
    title: '客户端配置',
    description: '把 Sub2API 接入常用的 AI 编程与桌面客户端。',
    order: 20,
  },
  {
    id: 'troubleshooting',
    title: '故障排查',
    description: '根据状态码和最小请求快速定位配置或服务问题。',
    order: 30,
  },
]

export const wikiArticles: readonly WikiArticle[] = [
  {
    slug: 'first-openai-request',
    section: 'getting-started',
    title: '第一次 OpenAI 兼容请求',
    summary: '用一个最小 curl 请求验证站点地址、API Key 和可用模型。',
    order: 10,
    tags: ['curl', 'OpenAI', 'API Key', '模型'],
    status: 'verified',
    lastVerified: '2026-09-03',
    verifiedWith: 'Sub2API /v1/models 路由',
    source: firstRequestSource,
  },
  {
    slug: 'cc-switch',
    section: 'clients',
    title: '使用 CC Switch 接入 Codex',
    summary: '从创建 API Key 到导入 CC Switch，完成 Codex 的首次请求验证。',
    order: 10,
    tags: ['CC Switch', 'Codex', 'API Key'],
    status: 'verified',
    lastVerified: '2026-09-05',
    verifiedWith: 'Sub2API API 密钥页面与 CC Switch Codex 导入链路',
    source: ccSwitchSource,
  },
  {
    slug: 'open-minis',
    section: 'clients',
    title: '使用 Open Minis 接入 Sub2API',
    summary: '在 iOS 或 Android 上填写 Base URL 和 API Key，刷新模型并选择推荐模型。',
    order: 20,
    tags: ['Open Minis', '移动端', 'iOS', 'Android', 'API Key'],
    status: 'verified',
    lastVerified: '2026-09-05',
    verifiedWith: 'Open Minis 官方下载入口、提供的设置截图与 Sub2API OpenAI 兼容接口',
    source: openMinisSource,
  },
  {
    slug: 'codex-image2',
    section: 'clients',
    title: '修复 Codex 的 gpt-image-2 调用',
    summary: '把完整提示词交给 Codex，自动安装 Skill 并安全复用 CC Switch 当前 provider。',
    order: 30,
    tags: ['Codex', 'gpt-image-2', 'Skill', 'CC Switch', '生图'],
    status: 'verified',
    lastVerified: '2026-09-05',
    verifiedWith: 'codex-image2 上游仓库、本地 Skill 与 CC Switch 动态 provider 适配提示词',
    source: codexImage2Source,
  },
  {
    slug: 'request-errors',
    section: 'troubleshooting',
    title: '请求失败时先看这里',
    summary: '从 HTTP 状态码、错误正文和最小请求开始定位问题。',
    order: 10,
    tags: ['401', '403', '404', '429', '5xx', '超时'],
    status: 'verified',
    lastVerified: '2026-09-03',
    verifiedWith: 'Sub2API 网关路由与错误处理',
    source: requestErrorsSource,
  },
]

export function wikiArticlePath(article: Pick<WikiArticle, 'section' | 'slug'>): string {
  return `/wiki/${article.section}/${article.slug}`
}

export function getWikiArticle(section: string, slug: string): WikiArticle | undefined {
  return wikiArticles.find((article) => article.section === section && article.slug === slug)
}

export function validateWikiArticles(articles: readonly WikiArticle[]): string[] {
  const errors: string[] = []
  const paths = new Set<string>()
  const sectionIds = new Set(wikiSections.map((section) => section.id))

  for (const article of articles) {
    const path = wikiArticlePath(article)
    if (paths.has(path)) {
      errors.push(`Duplicate Wiki path: ${path}`)
    }
    paths.add(path)

    if (!sectionIds.has(article.section)) {
      errors.push(`Unknown Wiki section: ${article.section}`)
    }
    if (!article.title.trim() || !article.summary.trim() || !article.source.trim()) {
      errors.push(`Incomplete Wiki article: ${path}`)
    }
    if (article.status === 'verified' && !article.lastVerified) {
      errors.push(`Verified Wiki article is missing lastVerified: ${path}`)
    }
  }

  return errors
}

const registryErrors = validateWikiArticles(wikiArticles)
if (registryErrors.length > 0) {
  throw new Error(registryErrors.join('\n'))
}

function normalizedSearchText(value: string): string {
  return value.normalize('NFKC').toLocaleLowerCase().replace(/\s+/g, ' ').trim()
}

export function searchWikiArticles(query: string): WikiArticle[] {
  const terms = normalizedSearchText(query).split(' ').filter(Boolean)
  if (terms.length === 0) {
    return [...wikiArticles]
  }

  return wikiArticles
    .map((article) => {
      const title = normalizedSearchText(article.title)
      const tags = normalizedSearchText(article.tags.join(' '))
      const summary = normalizedSearchText(article.summary)
      const body = normalizedSearchText(article.source)
      let score = 0

      for (const term of terms) {
        if (title === term) score += 100
        else if (title.startsWith(term)) score += 60
        else if (title.includes(term)) score += 40
        if (tags.includes(term)) score += 25
        if (summary.includes(term)) score += 15
        if (body.includes(term)) score += 5
      }

      return { article, score }
    })
    .filter((result) => result.score > 0)
    .sort((a, b) => b.score - a.score || a.article.order - b.article.order)
    .map((result) => result.article)
}
