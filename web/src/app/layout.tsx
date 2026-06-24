import type { Metadata } from "next";
import { Geist, Noto_Sans_SC, JetBrains_Mono } from "next/font/google";
import { BrowserTitleTicker } from "@/components/browser-title-ticker";
import { GlobalAnnouncementDialog } from "@/components/global-announcement-dialog";
import { UIPreferencesSync } from "@/components/ui-preferences-sync";
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
  title: browserTitle,
  description: "通过前十大重仓股实时行情，计算基金预估涨跌幅。盘中实时更新，让您随时掌握基金动态。",
  keywords: ["基金估值", "实时行情", "公募基金", "基金净值", "投资工具"],
  authors: [{ name: "涨了多少团队" }],
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
    description: "盘中实时计算基金预估涨跌幅",
    type: "website",
    siteName: "涨了多少 · FundLive",
    locale: "zh_CN",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" data-theme="dark" suppressHydrationWarning>
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
        <a className="skip-link" href="#main-content">跳到主要内容</a>
        <GlobalAnnouncementDialog />
        {children}
      </body>
    </html>
  );
}
