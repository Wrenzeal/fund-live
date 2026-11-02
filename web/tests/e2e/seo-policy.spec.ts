import { expect, test } from '@playwright/test'
import { isSearchIndexingEnabled, publicPageMetadata } from '../../src/lib/site-metadata'

test('preview and explicit noindex cannot be reopened by page metadata', () => {
  const previous = { flag: process.env.SEO_NOINDEX, vercel: process.env.VERCEL_ENV }
  try {
    for (const [vercel, flag, expected] of [
      ['production', '', true], ['preview', '', false], ['development', '', false],
      ['', '', true], ['', 'true', false], ['production', 'true', false],
    ] as const) {
      if (vercel) process.env.VERCEL_ENV = vercel
      else delete process.env.VERCEL_ENV
      process.env.SEO_NOINDEX = flag
      expect(isSearchIndexingEnabled()).toBe(expected)
      expect(publicPageMetadata('示例', '说明', '/example').robots).toMatchObject({ index: expected })
    }
  } finally {
    if (previous.flag === undefined) delete process.env.SEO_NOINDEX
    else process.env.SEO_NOINDEX = previous.flag
    if (previous.vercel === undefined) delete process.env.VERCEL_ENV
    else process.env.VERCEL_ENV = previous.vercel
  }
})
