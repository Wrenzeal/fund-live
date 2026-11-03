import type { Page } from '@playwright/test'

/** Check rendered foreground/background contrast, including translucent ancestors. */
export async function readabilityIssues(page: Page, selector: string) {
  return page.evaluate(selector => {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 1
    const ctx = canvas.getContext('2d', { willReadFrequently: true })!
    type Color = { r: number; g: number; b: number; a: number }
    function color(value: string): Color {
      ctx.clearRect(0, 0, 1, 1)
      ctx.fillStyle = value
      ctx.fillRect(0, 0, 1, 1)
      const [r, g, b, alpha] = ctx.getImageData(0, 0, 1, 1).data
      return { r, g, b, a: alpha / 255 }
    }
    function over(fg: Color, bg: Color): Color {
      return { r: fg.r * fg.a + bg.r * (1 - fg.a), g: fg.g * fg.a + bg.g * (1 - fg.a), b: fg.b * fg.a + bg.b * (1 - fg.a), a: 1 }
    }
    function luminance(c: Color) {
      const [r, g, b] = [c.r, c.g, c.b].map(v => v / 255).map(v => v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4)
      return .2126 * r + .7152 * g + .0722 * b
    }
    return [...document.querySelectorAll<HTMLElement>(selector)].flatMap(element => {
      const rect = element.getBoundingClientRect()
      const css = getComputedStyle(element)
      if (!rect.width || !rect.height || css.visibility === 'hidden' || element.matches(':disabled') || !element.innerText?.trim()) return []
      const parents: HTMLElement[] = []
      for (let node: HTMLElement | null = element; node; node = node.parentElement) {
        if (Number(getComputedStyle(node).opacity) < .95) return []
        parents.push(node)
      }
      let background = color(getComputedStyle(document.documentElement).backgroundColor)
      for (const node of parents.reverse()) background = over(color(getComputedStyle(node).backgroundColor), background)
      const foreground = over(color(css.color), background)
      const a = luminance(foreground), b = luminance(background)
      const ratio = (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
      const large = parseFloat(css.fontSize) >= 24 || parseFloat(css.fontSize) >= 18.66 && Number(css.fontWeight) >= 700
      return ratio + .01 >= (large ? 3 : 4.5) ? [] : [{ text: element.innerText.trim().slice(0, 65), ratio: Number(ratio.toFixed(2)), color: css.color, background: css.backgroundColor }]
    })
  }, selector)
}
