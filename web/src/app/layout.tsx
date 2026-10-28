import type { Metadata } from "next";
import { Geist, Noto_Sans_SC, JetBrains_Mono } from "next/font/google";
import { BrowserTitleTicker } from "@/components/browser-title-ticker";
import { GlobalAnnouncementDialog } from "@/components/global-announcement-dialog";
import { UIPreferencesSync } from "@/components/ui-preferences-sync";
import { SITE_DESCRIPTION, SITE_NAME, SITE_URL } from "@/lib/site-metadata";
import "./globals.css";

const geist = Geist({
  variable: "--font-geist",
  subsets: ["latin"],
});

const notoSansSC = Noto_Sans_SC({
  variable: "--font-noto-sans-sc",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
});

const browserTitle = "FundLive - 你的基金估值系统";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: { default: browserTitle, template: "%s | 涨了多少 FundLive" },
  description: SITE_DESCRIPTION,
  alternates: { canonical: "/" },
  robots: { index: true, follow: true },
  authors: [{ name: "涨了多少团队" }],
  verification: {
    google: process.env.GOOGLE_SITE_VERIFICATION,
    other: {
      ...(process.env.BING_SITE_VERIFICATION ? { "msvalidate.01": process.env.BING_SITE_VERIFICATION } : {}),
      ...(process.env.BAIDU_SITE_VERIFICATION ? { "baidu-site-verification": process.env.BAIDU_SITE_VERIFICATION } : {}),
    },
  },
  icons: {
    icon: [
      { url: "/favicon.svg", type: "image/svg+xml" },
      { url: "/favicon.ico", sizes: "any" },
    ],
    shortcut: "/favicon.ico",
    apple: "/favicon.svg",
  },
  openGraph: {
    title: browserTitle,
    description: SITE_DESCRIPTION,
    type: "website",
    url: SITE_URL,
    siteName: SITE_NAME,
    locale: "zh_CN",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" data-theme="classic" data-scroll-behavior="smooth" suppressHydrationWarning>
      <head>
        <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
        <link rel="alternate icon" href="/favicon.ico" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </head>
      <body
        className={`${geist.variable} ${notoSansSC.variable} ${jetbrainsMono.variable} antialiased`}
      >
        <UIPreferencesSync />
        <BrowserTitleTicker title={browserTitle} />
        <script
          id="website-structured-data"
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify({
            '@context': 'https://schema.org',
            '@type': 'WebSite',
            '@id': `${SITE_URL}/#website`,
            url: SITE_URL,
            name: SITE_NAME,
            alternateName: ['FundLive', '涨了多少'],
            description: SITE_DESCRIPTION,
            inLanguage: 'zh-CN',
          }).replace(/</g, '\\u003c') }}
        />
        <a className="skip-link" href="#main-content">跳到主要内容</a>
        <GlobalAnnouncementDialog />
        {children}
      </body>
    </html>
  );
}
