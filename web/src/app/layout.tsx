import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "FundLive - 实时基金估值",
  description: "通过前十大重仓股实时行情，计算基金预估涨跌幅。盘中实时更新，让您随时掌握基金动态。",
  keywords: ["基金估值", "实时行情", "公募基金", "基金净值", "投资工具"],
  authors: [{ name: "FundLive Team" }],
  openGraph: {
    title: "FundLive - 实时基金估值",
    description: "盘中实时计算基金预估涨跌幅",
    type: "website",
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
        <link rel="icon" href="/favicon.ico" />
        <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1" />
      </head>
      <body
        className={`${inter.variable} ${jetbrainsMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
