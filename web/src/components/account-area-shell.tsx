"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { ArrowUp, ChevronRight, Layers3, Sparkles, Wallet } from "lucide-react";
import { BrandMark } from "@/components/brand-mark";
import { HeaderFundSearch } from "@/components/header-fund-search";
import { ThemeSwitcher } from "@/components/theme-switcher";
import { UserAccountMenu } from "@/components/user-account-menu";
import { SiteFooter } from "@/components/site-footer";
import { useMobileTopSection } from "@/hooks/use-mobile-top-section";
import { useUIPreferences } from "@/hooks/use-ui-preferences";
import { cn } from "@/lib/utils";

interface AccountAreaShellProps {
  title: string;
  description: string;
  children: React.ReactNode;
}

const tabs = [
  { href: "/watchlist", label: "我的自选", icon: Layers3 },
  { href: "/holdings", label: "持仓明细", icon: Wallet },
];

const MOBILE_BREAKPOINT_QUERY = "(min-width: 768px)";
const MIN_ANCHOR_OFFSET_PX = 112;
const ANCHOR_OFFSET_GAP_PX = 16;

export function AccountAreaShell({
  title,
  description,
  children,
}: AccountAreaShellProps) {
  const pathname = usePathname();
  const { themeType, setThemeType, viewMode, setViewMode } = useUIPreferences();
  const { isAtTop, showBackToTop, scrollToTop } = useMobileTopSection();
  const headerRef = useRef<HTMLElement | null>(null);
  const topBarRef = useRef<HTMLDivElement | null>(null);
  const [anchorOffsetPx, setAnchorOffsetPx] = useState(MIN_ANCHOR_OFFSET_PX);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const updateAnchorOffset = () => {
      const headerHeight =
        headerRef.current?.getBoundingClientRect().height ?? 0;
      const topBarHeight =
        topBarRef.current?.getBoundingClientRect().height ?? 0;
      const isDesktop = window.matchMedia(MOBILE_BREAKPOINT_QUERY).matches;
      const measuredHeight = isDesktop
        ? headerHeight
        : topBarHeight || headerHeight;
      const nextOffset = Math.max(
        Math.ceil(measuredHeight + ANCHOR_OFFSET_GAP_PX),
        MIN_ANCHOR_OFFSET_PX,
      );
      setAnchorOffsetPx(nextOffset);
    };

    updateAnchorOffset();

    const resizeObserver = new ResizeObserver(() => {
      updateAnchorOffset();
    });

    if (headerRef.current) {
      resizeObserver.observe(headerRef.current);
    }
    if (topBarRef.current) {
      resizeObserver.observe(topBarRef.current);
    }

    window.addEventListener("resize", updateAnchorOffset);
    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", updateAnchorOffset);
    };
  }, []);

  const shellStyle = {
    "--account-shell-anchor-offset": `${anchorOffsetPx}px`,
  } as CSSProperties;

  return (
    <div className="min-h-[100dvh]" style={shellStyle}>
      <header
        ref={headerRef}
        className="sticky top-0 z-50 border-b border-[var(--card-border)] glass-strong"
      >
        <div className="container mx-auto px-4 py-4">
          <div
            ref={topBarRef}
            className="flex items-center justify-between gap-4"
          >
            <div className="flex items-center gap-4">
              <BrandMark subtitle="用户模块" />

              <div className="hidden items-center gap-2 text-sm text-theme-muted lg:flex">
                <ChevronRight className="h-4 w-4" />
                <span>{title}</span>
              </div>
            </div>

            <div className="hidden max-w-md flex-1 md:block">
              <HeaderFundSearch />
            </div>

            <div className="flex items-center gap-3">
              <UserAccountMenu />
              <ThemeSwitcher
                themeType={themeType}
                setThemeType={setThemeType}
                viewMode={viewMode}
                setViewMode={setViewMode}
                hideViewMode
              />
            </div>
          </div>

          <div
            className={cn(
              "overflow-hidden transition-all duration-300 md:overflow-visible md:transition-none",
              isAtTop
                ? "mt-4 max-h-[32rem] opacity-100"
                : "mt-0 max-h-0 opacity-0 pointer-events-none md:pointer-events-auto",
              "md:mt-4 md:max-h-none md:opacity-100",
            )}
          >
            <div className="md:hidden">
              <HeaderFundSearch />
            </div>

            <div className="mt-5 flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
              <div className="max-w-3xl space-y-3">
                <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/25 bg-cyan-500/10 px-4 py-2 text-xs tracking-[0.3em] text-cyan-300">
                  <Sparkles className="h-3.5 w-3.5" />
                  我的空间
                </div>
                <div>
                  <h1 className="text-3xl font-black text-theme-primary sm:text-4xl">
                    {title}
                  </h1>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-theme-secondary sm:text-base">
                    {description}
                  </p>
                </div>
              </div>

              <nav className="flex flex-wrap gap-3">
                {tabs.map((tab) => {
                  const Icon = tab.icon;
                  const active = pathname === tab.href;
                  return (
                    <Link
                      key={tab.href}
                      href={tab.href}
                      className={cn(
                        "group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border px-4 py-2.5 text-sm transition-all duration-200",
                        "hover:-translate-y-0.5 active:scale-[0.985]",
                        active
                          ? "account-standard-tab account-standard-tab-active border-cyan-500/40 bg-cyan-500/15 text-cyan-300 shadow-[0_14px_28px_rgba(34,211,238,0.14)]"
                          : "account-standard-tab account-standard-tab-idle border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-400/35 hover:bg-cyan-400/10 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)]",
                      )}
                    >
                      <span className="account-tab-shine" />
                      <span
                        className={cn(
                          "relative z-10 flex items-center gap-2",
                          active && "account-tab-active",
                        )}
                      >
                        <Icon
                          className={cn(
                            "h-4 w-4 transition-transform duration-300",
                            active
                              ? "scale-105"
                              : "group-hover:-rotate-6 group-hover:scale-110",
                          )}
                        />
                        {tab.label}
                      </span>
                    </Link>
                  );
                })}
              </nav>
            </div>
          </div>
        </div>
      </header>

      <main id="main-content" className="container mx-auto px-4 py-8">
        {children}
      </main>

      <SiteFooter compact />

      {showBackToTop && (
        <button
          type="button"
          onClick={scrollToTop}
          className="fixed bottom-5 right-4 z-50 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-[var(--card-bg)]/95 px-4 py-3 text-sm font-medium text-theme-primary shadow-[0_18px_36px_rgba(2,8,23,0.28)] backdrop-blur md:hidden"
          aria-label="回到顶部"
        >
          <ArrowUp className="h-4 w-4" />
          顶部
        </button>
      )}
    </div>
  );
}
