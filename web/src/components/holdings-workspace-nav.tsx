import {
  BarChart4,
  BellRing,
  ClipboardList,
  HeartPulse,
  History,
  LoaderCircle,
  Wallet,
  type LucideIcon,
} from "lucide-react";

import { Surface } from "@/components/ui/surface";
import { cn } from "@/lib/utils";

export type HoldingWorkspaceTab =
  | "summary"
  | "record"
  | "list"
  | "risk"
  | "ledger"
  | "tools";

interface WorkspaceTabConfig {
  id: HoldingWorkspaceTab;
  label: string;
  icon: LucideIcon;
}

interface QuickActionConfig {
  label: string;
  tab: HoldingWorkspaceTab;
  primary?: boolean;
}

interface HoldingsWorkspaceNavProps {
  holdingCount: number;
  activeTab: HoldingWorkspaceTab;
  detailText: string;
  isSeedingDemo: boolean;
  onTabChange: (tab: HoldingWorkspaceTab) => void;
  onSeedDemo: () => void;
}

const workspaceTabs: WorkspaceTabConfig[] = [
  {
    id: "summary",
    label: "总览",
    icon: Wallet,
  },
  {
    id: "record",
    label: "记录",
    icon: ClipboardList,
  },
  {
    id: "list",
    label: "持仓",
    icon: BarChart4,
  },
  { id: "risk", label: "风险", icon: HeartPulse },
  {
    id: "ledger",
    label: "流水",
    icon: History,
  },
  { id: "tools", label: "工具", icon: BellRing },
];

const quickActions: QuickActionConfig[] = [
  { label: "记一笔", tab: "record", primary: true },
  { label: "看持仓", tab: "list" },
  { label: "查风险", tab: "risk" },
  { label: "看流水", tab: "ledger" },
];

export function HoldingsWorkspaceNav({
  holdingCount,
  activeTab,
  detailText,
  isSeedingDemo,
  onTabChange,
  onSeedDemo,
}: HoldingsWorkspaceNavProps) {
  return (
    <Surface as="section" radius="xl" padding="md">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="text-sm font-medium text-theme-muted">持仓</div>
          <div className="mt-1 text-3xl font-black text-theme-primary">
            {holdingCount} 条记录
          </div>
          <div className="mt-2 text-xs text-theme-muted">{detailText}</div>
        </div>

        <div className="flex flex-wrap gap-2">
          {quickActions.map((action) => (
            <button
              key={action.tab}
              type="button"
              onClick={() => onTabChange(action.tab)}
              className={cn(
                "rounded-2xl border px-4 py-2 text-sm font-medium transition-colors",
                activeTab === action.tab
                  ? "border-cyan-300/45 bg-cyan-400/14 text-cyan-100 shadow-[0_12px_26px_rgba(34,211,238,0.12)]"
                  : action.primary
                    ? "border-cyan-300/30 bg-cyan-400/10 text-cyan-100 hover:border-cyan-200/45"
                    : "border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-300/35 hover:text-theme-primary",
              )}
            >
              {action.label}
            </button>
          ))}
          {holdingCount === 0 && (
            <button
              type="button"
              onClick={onSeedDemo}
              disabled={isSeedingDemo}
              className={cn(
                "inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-colors disabled:pointer-events-none disabled:opacity-60",
                "hover:border-[var(--accent-primary)] hover:text-theme-primary",
              )}
            >
              {isSeedingDemo ? (
                <LoaderCircle className="h-4 w-4 animate-spin" />
              ) : (
                <BarChart4 className="h-4 w-4" />
              )}
              <span>{isSeedingDemo ? "准备中…" : "快速开始"}</span>
            </button>
          )}
        </div>
      </div>

      {holdingCount > 0 && (
        <div className="mt-5 flex flex-wrap gap-2">
          {workspaceTabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => onTabChange(tab.id)}
                className={cn(
                  "rounded-xl border px-3 py-2 text-left transition-colors",
                  activeTab === tab.id
                    ? "border-cyan-300/45 bg-cyan-400/14 text-cyan-100 shadow-[0_14px_30px_rgba(34,211,238,0.12)]"
                    : "border-[var(--card-border)] bg-[var(--card-bg)]/56 text-theme-secondary hover:border-cyan-300/30 hover:text-theme-primary",
                )}
              >
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Icon className="h-4 w-4" />
                  {tab.label}
                </div>
              </button>
            );
          })}
        </div>
      )}
    </Surface>
  );
}
