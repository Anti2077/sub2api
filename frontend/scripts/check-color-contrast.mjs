const colors = {
  white: '#ffffff',
  gray50: '#f9fafb',
  gray100: '#f3f4f6',
  gray500: '#6b7280',
  gray600: '#4b5563',
  gray700: '#374151',
  gray800: '#1f2937',
  gray950: '#030712',
  dark100: '#f1f5f9',
  dark200: '#e2e8f0',
  dark300: '#cbd5e1',
  dark400: '#94a3b8',
  dark800: '#1e293b',
  dark900: '#0f172a',
  dark950: '#020617',
  primary50: '#f0fdfa',
  primary100: '#ccfbf1',
  primary400: '#2dd4bf',
  primary300: '#5eead4',
  primary700: '#0f766e',
  primary800: '#115e59',
  primary900: '#134e4a',
  emerald100: '#d1fae5',
  emerald400: '#34d399',
  emerald300: '#6ee7b7',
  emerald700: '#047857',
  emerald800: '#065f46',
  emerald900: '#064e3b',
  amber100: '#fef3c7',
  amber400: '#fbbf24',
  amber300: '#fcd34d',
  amber700: '#b45309',
  amber800: '#92400e',
  amber900: '#78350f',
  red100: '#fee2e2',
  red400: '#f87171',
  red300: '#fca5a5',
  red600: '#dc2626',
  red700: '#b91c1c',
  red800: '#991b1b',
  red900: '#7f1d1d',
  stripe: '#635bff',
  stripeHover: '#5851ea',
  airwallexLight: '#14171a',
  airwallexLightHover: '#20252a',
  airwallexDark: '#7af0c4',
  airwallexDarkHover: '#62d9ad',
  alipay: '#00aeef',
  alipayHover: '#009dd6',
  alipayActive: '#008cbe',
  wxpay: '#2bb741',
  wxpayHover: '#24a038',
  wxpayActive: '#1d8a2f'
}

function parseHex(hex) {
  const normalized = hex.replace('#', '')
  const value = normalized.length === 3
    ? normalized.split('').map((part) => `${part}${part}`).join('')
    : normalized

  if (!/^[0-9a-f]{6}$/i.test(value)) {
    throw new Error(`Invalid hex color: ${hex}`)
  }

  return [0, 2, 4].map((offset) => Number.parseInt(value.slice(offset, offset + 2), 16) / 255)
}

function relativeLuminance(hex) {
  const [red, green, blue] = parseHex(hex).map((channel) => (
    channel <= 0.03928
      ? channel / 12.92
      : ((channel + 0.055) / 1.055) ** 2.4
  ))

  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

function contrastRatio(foreground, background) {
  const foregroundLuminance = relativeLuminance(foreground)
  const backgroundLuminance = relativeLuminance(background)
  const lighter = Math.max(foregroundLuminance, backgroundLuminance)
  const darker = Math.min(foregroundLuminance, backgroundLuminance)

  return (lighter + 0.05) / (darker + 0.05)
}

function blend(foreground, background, opacity) {
  const foregroundChannels = parseHex(foreground)
  const backgroundChannels = parseHex(background)
  const channels = foregroundChannels.map((channel, index) => (
    Math.round((channel * opacity + backgroundChannels[index] * (1 - opacity)) * 255)
  ))

  return `#${channels.map((channel) => channel.toString(16).padStart(2, '0')).join('')}`
}

const checks = [
  ['body.light', 'gray800', 'gray50'],
  ['body.dark', 'dark100', 'dark950'],
  ['input.text.light', 'gray800', 'white'],
  ['input.text.dark', 'dark100', 'dark800'],
  ['input.placeholder.light', 'gray500', 'white'],
  ['input.placeholder.dark', 'dark300', 'dark800'],
  ['button.primary.start', 'white', 'primary700'],
  ['button.primary.end', 'white', 'primary800'],
  ['button.primary.hover-end', 'white', 'primary900'],
  ['button.danger.start', 'white', 'red600'],
  ['button.danger.end', 'white', 'red700'],
  ['button.success.start', 'white', 'emerald700'],
  ['button.success.end', 'white', 'emerald800'],
  ['button.warning.start', 'white', 'amber700'],
  ['button.warning.end', 'white', 'amber800'],
  ['button.stripe', 'white', 'stripe'],
  ['button.stripe.hover', 'white', 'stripeHover'],
  ['button.airwallex.light', 'white', 'airwallexLight'],
  ['button.airwallex.light-hover', 'white', 'airwallexLightHover'],
  ['button.airwallex.dark', 'gray950', 'airwallexDark'],
  ['button.airwallex.dark-hover', 'gray950', 'airwallexDarkHover'],
  ['button.alipay', 'gray950', 'alipay'],
  ['button.alipay.hover', 'gray950', 'alipayHover'],
  ['button.alipay.active', 'gray950', 'alipayActive'],
  ['button.wxpay', 'gray950', 'wxpay'],
  ['button.wxpay.hover', 'gray950', 'wxpayHover'],
  ['button.wxpay.active', 'gray950', 'wxpayActive'],
  ['text.sidebar-section.light', 'gray500', 'white'],
  ['text.sidebar-section.dark', 'dark400', 'dark900'],
  ['text.error.light', 'red700', 'white'],
  ['text.code.light', 'primary800', 'gray100'],
  ['text.code.dark', 'primary300', 'dark800'],
  ['badge.primary.light', 'primary700', 'primary100'],
  ['badge.success.light', 'emerald700', 'emerald100'],
  ['badge.warning.light', 'amber700', 'amber100'],
  ['badge.danger.light', 'red700', 'red100'],
  ['stat-icon.primary.dark', 'primary300', blend(colors.primary900, colors.dark800, 0.3)],
  ['stat-icon.success.dark', 'emerald300', blend(colors.emerald900, colors.dark800, 0.3)],
  ['stat-icon.warning.dark', 'amber300', blend(colors.amber900, colors.dark800, 0.3)],
  ['stat-icon.danger.dark', 'red300', blend(colors.red900, colors.dark800, 0.3)],
  ['badge.primary.dark', 'primary400', blend(colors.primary900, colors.dark800, 0.3)],
  ['badge.success.dark', 'emerald400', blend(colors.emerald900, colors.dark800, 0.3)],
  ['badge.warning.dark', 'amber400', blend(colors.amber900, colors.dark800, 0.3)],
  ['badge.danger.dark', 'red400', blend(colors.red900, colors.dark800, 0.3)]
]

const minimumRatio = 4.5
const failures = []

for (const [name, foregroundToken, backgroundToken] of checks) {
  const foreground = colors[foregroundToken] ?? foregroundToken
  const background = colors[backgroundToken] ?? backgroundToken
  const ratio = contrastRatio(foreground, background)
  const passes = ratio >= minimumRatio

  console.log(`${passes ? 'PASS' : 'FAIL'} ${ratio.toFixed(2).padStart(5)}:1 ${name} (${foreground} on ${background})`)

  if (!passes) {
    failures.push({ name, ratio })
  }
}

if (failures.length > 0) {
  console.error(`\n${failures.length} contrast check(s) failed. Target for normal text: ${minimumRatio}:1.`)
  process.exitCode = 1
} else {
  console.log(`\nAll ${checks.length} contrast checks passed at ${minimumRatio}:1 or higher.`)
}
