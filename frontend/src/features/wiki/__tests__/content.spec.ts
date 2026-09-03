import { describe, expect, it } from 'vitest'
import {
  searchWikiArticles,
  validateWikiArticles,
  wikiArticlePath,
  wikiArticles,
  type WikiArticle,
} from '../content'

describe('Wiki content registry', () => {
  it('contains only valid unique articles', () => {
    expect(validateWikiArticles(wikiArticles)).toEqual([])
    expect(new Set(wikiArticles.map(wikiArticlePath)).size).toBe(wikiArticles.length)
  })

  it('ranks title matches before body-only matches', () => {
    const results = searchWikiArticles('CC Switch')
    expect(results[0]?.slug).toBe('cc-switch')
  })

  it('finds articles by status-code tags', () => {
    expect(searchWikiArticles('429').map((article) => article.slug)).toContain('request-errors')
  })

  it('rejects duplicate paths and incomplete verified metadata', () => {
    const duplicate = { ...wikiArticles[0], lastVerified: null } as WikiArticle
    expect(validateWikiArticles([wikiArticles[0], duplicate])).toEqual([
      `Duplicate Wiki path: ${wikiArticlePath(duplicate)}`,
      `Verified Wiki article is missing lastVerified: ${wikiArticlePath(duplicate)}`,
    ])
  })
})
