import { baseCompile } from '@intlify/message-compiler'
import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'

describe('translation HTML security', () => {
  it.each([
    '<iframe srcdoc="<b>nested document</b>"></iframe>',
    '<iframe srcdoc="{content}"></iframe>'
  ])('neutralizes srcdoc when escaped translations are rendered as HTML: %s', message => {
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      const i18n = createI18n({
        legacy: false,
        locale: 'en',
        escapeParameter: true,
        warnHtmlMessage: false,
        messages: { en: { message: baseCompile(message, { jit: true }).ast } }
      })
      const html = i18n.global.t('message', { content: '<b>untrusted parameter</b>' })
      const container = document.createElement('template')
      container.innerHTML = html
      expect(container.content.querySelector('iframe')).not.toBeNull()
      expect(container.content.querySelector('[srcdoc]')).toBeNull()
    } finally {
      warning.mockRestore()
    }
  })
})
