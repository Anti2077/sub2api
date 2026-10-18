import ccSwitchSource from '@/content/wiki/zh/clients/cc-switch.md?raw'
import claudeCodeSource from '@/content/wiki/zh/clients/claude-code.md?raw'
import codexImageGenSource from '@/content/wiki/zh/clients/codex-image2.md?raw'
import dailyUseSource from '@/content/wiki/zh/getting-started/daily-use.md?raw'
import firstRequestSource from '@/content/wiki/zh/getting-started/first-request.md?raw'
import grokBuildSource from '@/content/wiki/zh/clients/grok-build.md?raw'
import openMinisSource from '@/content/wiki/zh/clients/open-minis.md?raw'
import proxyDirectRoutingSource from '@/content/wiki/zh/troubleshooting/proxy-direct-routing.md?raw'
import requestErrorsSource from '@/content/wiki/zh/troubleshooting/request-errors.md?raw'
import rustdeskSource from '@/content/wiki/zh/clients/rustdesk.md?raw'

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
    slug: 'daily-use',
    section: 'getting-started',
    title: '本站用户日常使用指南',
    summary: '人工充值、余额、用量、使用记录、可用渠道和渠道状态的日常操作说明。',
    order: 20,
    tags: ['充值', '余额', '用量', '使用记录', '可用渠道', '渠道状态', 'API Key'],
    status: 'verified',
    lastVerified: '2026-09-05',
    verifiedWith: 'Sub2API 普通用户页面与本站人工充值流程',
    source: dailyUseSource,
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
    title: '使用 codex-image-gen++ 在 Codex 中生成和编辑图片',
    summary: '安装 MCP、用 init 导入 CC Switch 或手动配置，交给 Codex 选择模型并保存图片。',
    order: 30,
    tags: ['Codex', 'codex-image-gen++', 'MCP', 'CC Switch', '生图', '图片编辑'],
    status: 'verified',
    lastVerified: '2026-10-18',
    verifiedWith: 'codex-image-gen++ v0.2.0 源码、发布安装包与 Windows/macOS/Linux Node.js 22/24 CI；权限以真实请求为准',
    source: codexImageGenSource,
  },
  {
    slug: 'claude-code',
    section: 'clients',
    title: '使用 CC Switch 接入 Claude Code',
    summary: '创建 Claude/Anthropic 分组密钥，导入 CC Switch 并完成 Claude Code 请求验证。',
    order: 40,
    tags: ['CC Switch', 'Claude Code', 'Anthropic', 'API Key'],
    status: 'verified',
    lastVerified: '2026-09-05',
    verifiedWith: 'Claude Code 官方安装文档、CC Switch Claude provider 与 Sub2API 导入实现',
    source: claudeCodeSource,
  },
  {
    slug: 'grok-build',
    section: 'clients',
    title: '使用 CC Switch 接入 Grok Build',
    summary: '安装开源 Grok Build，导入 Grok 分组密钥并验证模型与真实请求。',
    order: 50,
    tags: ['CC Switch', 'Grok Build', 'Grok', '开源', 'API Key'],
    status: 'verified',
    lastVerified: '2026-09-05',
    verifiedWith: 'Grok Build 官方仓库与文档、CC Switch v3.18.0、Sub2API 导入实现',
    source: grokBuildSource,
  },
  {
    slug: 'rustdesk',
    section: 'clients',
    title: '使用本站服务器配置 RustDesk',
    summary: '从蓝奏云下载 Windows、Mac 或 Linux 安装包，导入本站服务器配置并发送本机 ID。',
    order: 60,
    tags: ['RustDesk', '蓝奏云', '下载', 'Windows', 'macOS', 'Linux', '远程维护', 'ID 服务器', '中继服务器'],
    status: 'verified',
    lastVerified: '2026-10-13',
    verifiedWith: 'RustDesk 1.5.0 官方安装包 SHA-256 与蓝奏云分享页；配置步骤依据 2026-09-29 的 macOS 截图，截图版本未记录',
    source: rustdeskSource,
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
  {
    slug: 'proxy-direct-routing',
    section: 'troubleshooting',
    title: '网站无法访问：让 anti2077.xyz 走直连',
    summary: '在 Clash Verge Rev、Shadowrocket、Surge、Stash 或 sing-box 中，让本站根域名及其子域名不经过代理。',
    order: 20,
    tags: ['代理', '直连', 'DIRECT', 'Clash Verge Rev', 'Shadowrocket', 'Surge', 'Stash', 'sing-box', 'DNS', '超时'],
    status: 'verified',
    lastVerified: '2026-09-08',
    verifiedWith: 'Clash/Mihomo、Surge 与 sing-box 域名路由规则语义；本站前置域名',
    source: proxyDirectRoutingSource,
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
