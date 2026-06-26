"use client";

import { useEffect, useMemo, useRef, useState, type DragEvent } from "react";
import {
  Check,
  ChevronDown,
  FileSearch,
  FolderPlus,
  GripVertical,
  Layers3,
  LoaderCircle,
  Palette,
  PencilLine,
  Plus,
  Sparkles,
  Trash2,
  X,
} from "lucide-react";
import { AccountAreaShell } from "@/components/account-area-shell";
import { ActionButton } from "@/components/ui/action-button";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBanner } from "@/components/ui/status-banner";
import { Surface } from "@/components/ui/surface";
import { FloatingListbox } from "@/components/floating-listbox";
import { ScrollReveal, ScrollRevealStack } from "@/components/scroll-reveal";
import { WatchlistFundCard } from "@/components/watchlist-fund-card";
import { useCurrentUser } from "@/hooks/use-auth";
import { useFundAnalyses, useFundHistoryBatch, useFundSearch } from "@/hooks/use-fund-data";
import {
  useUserPortfolio,
  type WatchlistGroup,
} from "@/hooks/use-user-portfolio";
import {
  GROUP_ACCENT_OPTIONS,
  type WatchlistAccent,
  watchlistAccentBadgeClass,
  watchlistAccentLabel,
  watchlistAccentToClass,
} from "@/lib/watchlist-accent";
import { cn } from "@/lib/utils";

type GroupViewMode = "all" | "focused";
export default function WatchlistPage() {
  const { user, isLoading } = useCurrentUser();
  const {
    watchlistGroups,
    totalWatchlistFunds,
    seedDemoData,
    createGroup,
    updateGroup,
    reorderGroups,
    deleteGroup,
    addFundToGroup,
    removeFundFromGroup,
  } = useUserPortfolio(user?.id ?? null);

  const [groupName, setGroupName] = useState("");
  const [groupDescription, setGroupDescription] = useState("");
  const [selectedGroupID, setSelectedGroupID] = useState<string>("");
  const [focusedGroupId, setFocusedGroupId] = useState<string>("");
  const [groupViewMode, setGroupViewMode] = useState<GroupViewMode>("all");
  const [groupSearchQuery, setGroupSearchQuery] = useState("");
  const [collapsedGroupIds, setCollapsedGroupIds] = useState<Set<string>>(
    new Set(),
  );
  const [animatingNavGroupId, setAnimatingNavGroupId] = useState<string | null>(
    null,
  );
  const [animatingCollapseGroupId, setAnimatingCollapseGroupId] = useState<
    string | null
  >(null);
  const [animatingEditGroupId, setAnimatingEditGroupId] = useState<
    string | null
  >(null);
  const [animatingViewMode, setAnimatingViewMode] =
    useState<GroupViewMode | null>(null);
  const [isGroupMenuOpen, setIsGroupMenuOpen] = useState(false);
  const groupMenuButtonRef = useRef<HTMLButtonElement | null>(null);
  const [fundQuery, setFundQuery] = useState("");
  const [isFundSearchMenuOpen, setIsFundSearchMenuOpen] = useState(false);
  const fundSearchInputRef = useRef<HTMLInputElement | null>(null);
  const [isCreatingGroup, setIsCreatingGroup] = useState(false);
  const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
  const [editingGroupName, setEditingGroupName] = useState("");
  const [editingGroupDescription, setEditingGroupDescription] = useState("");
  const [editingGroupAccent, setEditingGroupAccent] =
    useState<WatchlistAccent>("cyan");
  const [savingGroupId, setSavingGroupId] = useState<string | null>(null);
  const [draggingGroupId, setDraggingGroupId] = useState<string | null>(null);
  const [dragOverGroupId, setDragOverGroupId] = useState<string | null>(null);
  const [isReorderingGroups, setIsReorderingGroups] = useState(false);
  const [localGroupOrderIds, setLocalGroupOrderIds] = useState<string[] | null>(
    null,
  );
  const [deletingGroupID, setDeletingGroupID] = useState<string | null>(null);
  const [watchlistFeedback, setWatchlistFeedback] = useState<{
    type: "success" | "error";
    message: string;
  } | null>(null);
  const groupSectionRefs = useRef<Record<string, HTMLElement | null>>({});
  const navAnimationTimerRef = useRef<number | null>(null);
  const collapseAnimationTimerRef = useRef<number | null>(null);
  const editAnimationTimerRef = useRef<number | null>(null);
  const viewModeAnimationTimerRef = useRef<number | null>(null);
  const { results, isLoading: isFundSearchLoading } = useFundSearch(fundQuery);
  const visibleFundResults = useMemo(() => results.slice(0, 5), [results]);
  const shouldShowFundSearchMenu =
    isFundSearchMenuOpen && fundQuery.trim().length >= 2;
  const watchlistFundIDs = useMemo(
    () =>
      Array.from(
        new Set(
          watchlistGroups
            .flatMap((group) => group.funds.map((item) => item.fund_id))
            .filter(Boolean),
        ),
      ),
    [watchlistGroups],
  );
  const { analysesByFundID } = useFundAnalyses(watchlistFundIDs);
  const { historiesByFundID, isLoading: isHistoryLoading } = useFundHistoryBatch(watchlistFundIDs, 15);

  const orderedWatchlistGroups = useMemo(() => {
    if (
      !localGroupOrderIds ||
      localGroupOrderIds.length !== watchlistGroups.length
    ) {
      return watchlistGroups;
    }

    const groupByID = new Map(
      watchlistGroups.map((group) => [group.id, group]),
    );
    const ordered = localGroupOrderIds
      .map((id) => groupByID.get(id) ?? null)
      .filter((group): group is WatchlistGroup => group !== null);

    if (ordered.length !== watchlistGroups.length) {
      return watchlistGroups;
    }

    return ordered;
  }, [localGroupOrderIds, watchlistGroups]);

  const selectedGroup = useMemo(
    () =>
      orderedWatchlistGroups.find((group) => group.id === selectedGroupID) ??
      null,
    [orderedWatchlistGroups, selectedGroupID],
  );
  const selectedGroupLabel = selectedGroup?.name || "选择一个分组";
  const normalizedGroupSearch = groupSearchQuery.trim().toLowerCase();

  const filteredGroups = useMemo(() => {
    if (!normalizedGroupSearch) {
      return orderedWatchlistGroups;
    }

    return orderedWatchlistGroups.filter((group) => {
      const name = group.name.toLowerCase();
      const description = (group.description || "").toLowerCase();
      return (
        name.includes(normalizedGroupSearch) ||
        description.includes(normalizedGroupSearch)
      );
    });
  }, [normalizedGroupSearch, orderedWatchlistGroups]);

  const focusedGroup = useMemo(
    () => filteredGroups.find((group) => group.id === focusedGroupId) ?? null,
    [filteredGroups, focusedGroupId],
  );

  const visibleGroups = useMemo(() => {
    if (groupViewMode === "focused") {
      return focusedGroup ? [focusedGroup] : [];
    }
    return filteredGroups;
  }, [filteredGroups, focusedGroup, groupViewMode]);

  useEffect(() => {
    if (watchlistGroups.length === 0) {
      setFocusedGroupId("");
      setSelectedGroupID("");
      return;
    }

    if (
      selectedGroupID &&
      !watchlistGroups.some((group) => group.id === selectedGroupID)
    ) {
      setSelectedGroupID("");
    }

    if (
      !focusedGroupId ||
      !watchlistGroups.some((group) => group.id === focusedGroupId)
    ) {
      setFocusedGroupId(watchlistGroups[0].id);
    }
  }, [focusedGroupId, selectedGroupID, watchlistGroups]);

  useEffect(() => {
    if (filteredGroups.length === 0 || groupViewMode !== "focused") {
      return;
    }

    if (!filteredGroups.some((group) => group.id === focusedGroupId)) {
      setFocusedGroupId(filteredGroups[0].id);
    }
  }, [filteredGroups, focusedGroupId, groupViewMode]);

  useEffect(() => {
    if (
      editingGroupId &&
      !watchlistGroups.some((group) => group.id === editingGroupId)
    ) {
      setEditingGroupId(null);
      setEditingGroupName("");
      setEditingGroupDescription("");
      setEditingGroupAccent("cyan");
      setSavingGroupId(null);
    }
  }, [editingGroupId, watchlistGroups]);

  useEffect(() => {
    if (!localGroupOrderIds) {
      return;
    }

    if (
      watchlistGroups.length === localGroupOrderIds.length &&
      watchlistGroups.every(
        (group, index) => group.id === localGroupOrderIds[index],
      )
    ) {
      setLocalGroupOrderIds(null);
    }
  }, [localGroupOrderIds, watchlistGroups]);

  useEffect(() => {
    return () => {
      if (navAnimationTimerRef.current !== null) {
        window.clearTimeout(navAnimationTimerRef.current);
      }
      if (collapseAnimationTimerRef.current !== null) {
        window.clearTimeout(collapseAnimationTimerRef.current);
      }
      if (editAnimationTimerRef.current !== null) {
        window.clearTimeout(editAnimationTimerRef.current);
      }
      if (viewModeAnimationTimerRef.current !== null) {
        window.clearTimeout(viewModeAnimationTimerRef.current);
      }
    };
  }, []);

  const reorderEnabled =
    groupViewMode === "all" &&
    normalizedGroupSearch === "" &&
    orderedWatchlistGroups.length > 1;

  const triggerNavAnimation = (groupID: string) => {
    if (navAnimationTimerRef.current !== null) {
      window.clearTimeout(navAnimationTimerRef.current);
    }
    setAnimatingNavGroupId(groupID);
    navAnimationTimerRef.current = window.setTimeout(() => {
      setAnimatingNavGroupId((current) =>
        current === groupID ? null : current,
      );
      navAnimationTimerRef.current = null;
    }, 560);
  };

  const triggerCollapseAnimation = (groupID: string) => {
    if (collapseAnimationTimerRef.current !== null) {
      window.clearTimeout(collapseAnimationTimerRef.current);
    }
    setAnimatingCollapseGroupId(groupID);
    collapseAnimationTimerRef.current = window.setTimeout(() => {
      setAnimatingCollapseGroupId((current) =>
        current === groupID ? null : current,
      );
      collapseAnimationTimerRef.current = null;
    }, 560);
  };

  const triggerEditAnimation = (groupID: string) => {
    if (editAnimationTimerRef.current !== null) {
      window.clearTimeout(editAnimationTimerRef.current);
    }
    setAnimatingEditGroupId(groupID);
    editAnimationTimerRef.current = window.setTimeout(() => {
      setAnimatingEditGroupId((current) =>
        current === groupID ? null : current,
      );
      editAnimationTimerRef.current = null;
    }, 560);
  };

  const triggerViewModeAnimation = (mode: GroupViewMode) => {
    if (viewModeAnimationTimerRef.current !== null) {
      window.clearTimeout(viewModeAnimationTimerRef.current);
    }
    setAnimatingViewMode(mode);
    viewModeAnimationTimerRef.current = window.setTimeout(() => {
      setAnimatingViewMode((current) => (current === mode ? null : current));
      viewModeAnimationTimerRef.current = null;
    }, 560);
  };

  const handleChangeGroupViewMode = (mode: GroupViewMode) => {
    triggerViewModeAnimation(mode);
    setGroupViewMode(mode);
  };

  const handleFocusGroup = (groupID: string) => {
    triggerNavAnimation(groupID);
    setFocusedGroupId(groupID);
    setCollapsedGroupIds((current) => {
      const next = new Set(current);
      next.delete(groupID);
      return next;
    });

    if (groupViewMode === "focused") {
      return;
    }

    window.requestAnimationFrame(() => {
      groupSectionRefs.current[groupID]?.scrollIntoView({
        behavior: "smooth",
        block: "start",
      });
    });
  };

  const toggleGroupCollapse = (groupID: string) => {
    triggerCollapseAnimation(groupID);
    setCollapsedGroupIds((current) => {
      const next = new Set(current);
      if (next.has(groupID)) {
        next.delete(groupID);
      } else {
        next.add(groupID);
      }
      return next;
    });
  };

  const handleCreateGroup = async () => {
    if (isCreatingGroup) {
      return;
    }

    setIsCreatingGroup(true);

    try {
      await createGroup(groupName, groupDescription);
      setGroupName("");
      setGroupDescription("");
    } finally {
      setIsCreatingGroup(false);
    }
  };

  const openGroupEditor = (group: WatchlistGroup) => {
    triggerEditAnimation(group.id);
    setWatchlistFeedback(null);
    setIsGroupMenuOpen(false);
    setIsFundSearchMenuOpen(false);
    setEditingGroupId(group.id);
    setEditingGroupName(group.name);
    setEditingGroupDescription(group.description || "");
    setEditingGroupAccent((group.accent as WatchlistAccent) || "cyan");
  };

  const closeGroupEditor = () => {
    setEditingGroupId(null);
    setEditingGroupName("");
    setEditingGroupDescription("");
    setEditingGroupAccent("cyan");
    setSavingGroupId(null);
  };

  const handleUpdateGroup = async (group: WatchlistGroup) => {
    if (savingGroupId) {
      return;
    }

    const normalizedName = editingGroupName.trim();
    const normalizedDescription = editingGroupDescription.trim();
    if (
      normalizedName === group.name.trim() &&
      normalizedDescription === (group.description || "").trim() &&
      editingGroupAccent === group.accent
    ) {
      closeGroupEditor();
      return;
    }

    setWatchlistFeedback(null);
    setSavingGroupId(group.id);

    try {
      await updateGroup(
        group.id,
        normalizedName,
        normalizedDescription,
        editingGroupAccent,
      );
      setWatchlistFeedback({
        type: "success",
        message: `已更新「${normalizedName}」的分组信息。`,
      });
      closeGroupEditor();
    } catch (error) {
      setWatchlistFeedback({
        type: "error",
        message:
          error instanceof Error
            ? error.message
            : "更新分组信息失败，请稍后重试。",
      });
      setSavingGroupId(null);
    }
  };

  const handleReorderGroups = async (nextOrderIds: string[]) => {
    if (isReorderingGroups) {
      return;
    }

    setWatchlistFeedback(null);
    setLocalGroupOrderIds(nextOrderIds);
    setIsReorderingGroups(true);

    try {
      await reorderGroups(nextOrderIds);
    } catch (error) {
      setLocalGroupOrderIds(null);
      setWatchlistFeedback({
        type: "error",
        message:
          error instanceof Error
            ? error.message
            : "调整分组顺序失败，请稍后重试。",
      });
    } finally {
      setIsReorderingGroups(false);
      setDraggingGroupId(null);
      setDragOverGroupId(null);
    }
  };

  const handleGroupDragStart = (
    event: DragEvent<HTMLElement>,
    groupID: string,
  ) => {
    if (!reorderEnabled) {
      return;
    }
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", groupID);
    setDraggingGroupId(groupID);
    setDragOverGroupId(groupID);
  };

  const handleGroupDragOver = (
    event: DragEvent<HTMLElement>,
    targetGroupID: string,
  ) => {
    if (
      !reorderEnabled ||
      !draggingGroupId ||
      draggingGroupId === targetGroupID
    ) {
      return;
    }

    event.preventDefault();
    setDragOverGroupId(targetGroupID);
  };

  const handleGroupDrop = async (targetGroupID: string) => {
    if (
      !reorderEnabled ||
      !draggingGroupId ||
      draggingGroupId === targetGroupID
    ) {
      setDraggingGroupId(null);
      setDragOverGroupId(null);
      return;
    }

    const currentIDs = orderedWatchlistGroups.map((group) => group.id);
    const draggingIndex = currentIDs.indexOf(draggingGroupId);
    const targetIndex = currentIDs.indexOf(targetGroupID);
    if (draggingIndex === -1 || targetIndex === -1) {
      setDraggingGroupId(null);
      setDragOverGroupId(null);
      return;
    }

    const nextOrder = [...currentIDs];
    const [moved] = nextOrder.splice(draggingIndex, 1);
    nextOrder.splice(targetIndex, 0, moved);
    await handleReorderGroups(nextOrder);
  };

  const handleDeleteGroup = async (groupID: string) => {
    if (deletingGroupID) {
      return;
    }

    setDeletingGroupID(groupID);

    try {
      await new Promise((resolve) => window.setTimeout(resolve, 180));
      await deleteGroup(groupID);
    } finally {
      setDeletingGroupID(null);
    }
  };

  if (isLoading) {
    return (
      <AccountAreaShell
        title="我的自选"
        description="按分组管理重点关注的基金。"
      >
        <Surface padding="none" radius="xl" className="h-64 animate-pulse">
          <span className="sr-only">正在加载自选分组</span>
        </Surface>
      </AccountAreaShell>
    );
  }

  if (!user) {
    return (
      <AccountAreaShell
        title="我的自选"
        description="按分组管理重点关注的基金。"
      >
        <EmptyState
          icon={<Layers3 className="h-10 w-10" />}
          title="登录后可查看我的自选"
          description="登录后可同步查看和管理你的分组与基金清单。"
          action={
            <div className="flex justify-center gap-3">
              <ActionButton href="/auth/login" variant="subtle">
                去登录
              </ActionButton>
              <ActionButton href="/auth/register" variant="primary">
                去注册
              </ActionButton>
            </div>
          }
        />
      </AccountAreaShell>
    );
  }

  return (
    <AccountAreaShell
      title="我的自选"
      description="按分组整理关注的基金，并快速查看每只基金的实时预估涨跌幅与迷你走势。"
    >
      <ScrollRevealStack className="space-y-8">
        {watchlistFeedback && (
          <StatusBanner
            tone={watchlistFeedback.type === "success" ? "success" : "warning"}
          >
            {watchlistFeedback.message}
          </StatusBanner>
        )}
        <div className="grid gap-6 xl:grid-cols-[1.4fr_0.9fr]">
          <Surface as="section" radius="xl" padding="md">
            <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <div className="text-sm text-theme-muted">分组总览</div>
                <div className="mt-1 text-3xl font-black text-theme-primary">
                  {watchlistGroups.length} 个分组 / {totalWatchlistFunds} 只基金
                </div>
              </div>

              {watchlistGroups.length === 0 && (
                <ActionButton
                  type="button"
                  variant="subtle"
                  onClick={() => void seedDemoData()}
                >
                  <Sparkles className="h-4 w-4" />
                  快速开始
                </ActionButton>
              )}
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm text-theme-secondary">新增分组</span>
                <input
                  value={groupName}
                  onChange={(event) => setGroupName(event.target.value)}
                  placeholder="例如：核心观察"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </label>

              <label className="space-y-2">
                <span className="text-sm text-theme-secondary">分组说明</span>
                <input
                  value={groupDescription}
                  onChange={(event) => setGroupDescription(event.target.value)}
                  placeholder="例如：长期定投、行业轮动等"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </label>
            </div>

            <ActionButton
              type="button"
              variant="primary"
              onClick={() => void handleCreateGroup()}
              disabled={isCreatingGroup}
              className={cn("mt-4", isCreatingGroup && "action-button-pop")}
            >
              {isCreatingGroup ? (
                <LoaderCircle className="h-4 w-4 animate-spin" />
              ) : (
                <FolderPlus className="h-4 w-4" />
              )}
              {isCreatingGroup ? "创建中..." : "创建分组"}
            </ActionButton>
          </Surface>

          <Surface as="section" radius="xl" padding="md">
            <div className="mb-5">
              <div className="text-sm text-theme-muted">把基金放进分组</div>
              <div className="mt-1 text-2xl font-bold text-theme-primary">
                分组管理
              </div>
            </div>

            <div className="space-y-4">
              <div className={cn("relative", isGroupMenuOpen && "z-[80]")}>
                <button
                  ref={groupMenuButtonRef}
                  type="button"
                  onClick={() => {
                    if (watchlistGroups.length === 0) return;
                    setIsGroupMenuOpen((open) => !open);
                  }}
                  disabled={watchlistGroups.length === 0}
                  className={cn(
                    "watchlist-select-shell group relative block w-full overflow-hidden rounded-[24px] border border-[var(--input-border)] px-4 py-3 text-left transition-all duration-200",
                    "hover:border-cyan-400/35 hover:bg-[var(--input-bg)] focus:outline-none focus-visible:border-cyan-400/55 focus-visible:bg-[var(--input-bg)] focus-visible:shadow-[0_14px_30px_rgba(34,211,238,0.12)]",
                    "disabled:cursor-not-allowed disabled:opacity-70",
                    isGroupMenuOpen &&
                      "border-cyan-400/55 bg-[var(--input-bg)] shadow-[0_14px_30px_rgba(34,211,238,0.12)]",
                  )}
                  aria-haspopup="listbox"
                  aria-expanded={isGroupMenuOpen}
                >
                  <span className="holding-picker-shine" />
                  <span className="relative z-10 block text-xs font-medium tracking-[0.18em] text-theme-muted">
                    目标分组
                  </span>
                  <div className="relative z-10 mt-3 flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div
                        className={cn(
                          "truncate text-sm font-medium",
                          selectedGroup
                            ? "text-theme-primary"
                            : "text-theme-secondary",
                        )}
                      >
                        {watchlistGroups.length === 0
                          ? "暂无可用分组"
                          : selectedGroupLabel}
                      </div>
                      <div className="mt-1 text-xs text-theme-muted">
                        {watchlistGroups.length === 0
                          ? "先在左侧创建分组，再把基金加入进去"
                          : selectedGroup
                            ? `当前分组共 ${selectedGroup.funds.length} 只基金`
                            : "先选择分组，再把搜索结果加入进去"}
                      </div>
                    </div>
                    <ChevronDown
                      className={cn(
                        "h-4 w-4 shrink-0 text-cyan-300 transition-all duration-300",
                        isGroupMenuOpen
                          ? "rotate-180"
                          : "group-hover:translate-y-0.5",
                      )}
                    />
                  </div>
                </button>

                <FloatingListbox
                  open={isGroupMenuOpen}
                  triggerRef={groupMenuButtonRef}
                  ariaLabel="目标分组"
                  withBackdrop
                  onClose={() => setIsGroupMenuOpen(false)}
                  gap={12}
                  maxHeight={288}
                >
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedGroupID("");
                      setIsGroupMenuOpen(false);
                    }}
                    className={cn(
                      "relative z-10 flex w-full items-start justify-between gap-3 rounded-[18px] px-4 py-3 text-left transition-colors",
                      !selectedGroup
                        ? "bg-cyan-500/14 text-cyan-100"
                        : "text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary",
                    )}
                    role="option"
                    aria-selected={!selectedGroup}
                  >
                    <div>
                      <div className="text-sm font-medium">暂不选择分组</div>
                      <div className="mt-1 text-xs text-theme-muted">
                        保留当前搜索结果，不立即加入任何分组
                      </div>
                    </div>
                    {!selectedGroup && (
                      <Check className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" />
                    )}
                  </button>

                  {watchlistGroups.map((group) => {
                    const active = group.id === selectedGroupID;
                    return (
                      <button
                        key={group.id}
                        type="button"
                        onClick={() => {
                          setSelectedGroupID(group.id);
                          setIsGroupMenuOpen(false);
                        }}
                        className={cn(
                          "relative z-10 flex w-full items-start justify-between gap-3 rounded-[18px] px-4 py-3 text-left transition-colors",
                          active
                            ? "bg-cyan-500/14 text-cyan-100"
                            : "text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary",
                        )}
                        role="option"
                        aria-selected={active}
                      >
                        <div className="min-w-0">
                          <div className="truncate text-sm font-medium">
                            {group.name}
                          </div>
                          <div className="mt-1 text-xs text-theme-muted">
                            {group.description || "未填写分组说明"} ·{" "}
                            {group.funds.length} 只基金
                          </div>
                        </div>
                        {active && (
                          <Check className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" />
                        )}
                      </button>
                    );
                  })}
                </FloatingListbox>
              </div>

              <div className="relative">
                <input
                  ref={fundSearchInputRef}
                  value={fundQuery}
                  onChange={(event) => {
                    setFundQuery(event.target.value);
                    setIsFundSearchMenuOpen(true);
                  }}
                  onFocus={() => setIsFundSearchMenuOpen(true)}
                  onBlur={() => {
                    window.setTimeout(
                      () => setIsFundSearchMenuOpen(false),
                      120,
                    );
                  }}
                  placeholder="搜索基金代码或名称"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                  role="combobox"
                  aria-haspopup="listbox"
                  aria-expanded={shouldShowFundSearchMenu}
                  aria-controls="watchlist-fund-search-results"
                  aria-autocomplete="list"
                />

                <FloatingListbox
                  id="watchlist-fund-search-results"
                  open={shouldShowFundSearchMenu}
                  triggerRef={fundSearchInputRef}
                  ariaLabel="基金搜索结果"
                  maxHeight={320}
                >
                  {isFundSearchLoading ? (
                    <div className="flex items-center gap-2 rounded-[18px] px-4 py-3 text-sm text-theme-secondary">
                      <LoaderCircle className="h-4 w-4 animate-spin text-cyan-300" />
                      正在搜索基金...
                    </div>
                  ) : visibleFundResults.length > 0 ? (
                    visibleFundResults.map((fund) => (
                      <button
                        key={fund.id}
                        type="button"
                        disabled={!selectedGroup}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => {
                          if (!selectedGroup) return;
                          void addFundToGroup(selectedGroup.id, fund.id);
                          setFundQuery("");
                          setIsFundSearchMenuOpen(false);
                        }}
                        className={cn(
                          "relative z-10 flex w-full items-center justify-between gap-3 rounded-[18px] px-4 py-3 text-left transition-colors",
                          selectedGroup
                            ? "text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary"
                            : "cursor-not-allowed text-theme-muted opacity-70",
                        )}
                        role="option"
                        aria-selected={false}
                      >
                        <div className="min-w-0">
                          <div className="truncate text-sm font-medium text-theme-primary">
                            {fund.name}
                          </div>
                          <div className="mt-1 text-xs text-theme-muted">
                            {fund.id}
                            {selectedGroup
                              ? ` · 加入 ${selectedGroup.name}`
                              : " · 请先选择目标分组"}
                          </div>
                        </div>
                        <Plus
                          className={cn(
                            "h-4 w-4 shrink-0",
                            selectedGroup
                              ? "text-cyan-300"
                              : "text-theme-muted",
                          )}
                        />
                      </button>
                    ))
                  ) : (
                    <div className="rounded-[18px] px-4 py-3 text-sm text-theme-secondary">
                      暂无匹配基金，请尝试基金代码或更完整名称。
                    </div>
                  )}
                </FloatingListbox>
              </div>
            </div>
          </Surface>
        </div>

        <div className="space-y-6">
          {watchlistGroups.length === 0 ? (
            <EmptyState
              icon={<Layers3 className="h-10 w-10" />}
              title="还没有自选分组"
              description="你可以先创建分组，再把基金加入对应的观察篮子；创建后的分组和基金会自动保存。"
            />
          ) : (
            <>
              <Surface as="section" radius="xl" padding="md">
                <div className="grid gap-4 xl:grid-cols-[1.1fr_1fr]">
                  <div className="space-y-3">
                    <div className="text-sm text-theme-muted">快速定位分组</div>
                    <div className="relative">
                      <FileSearch className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-theme-muted" />
                      <input
                        value={groupSearchQuery}
                        onChange={(event) =>
                          setGroupSearchQuery(event.target.value)
                        }
                        placeholder="搜索分组名称或说明"
                        className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] py-3 pl-11 pr-4 text-theme-primary outline-none placeholder:text-theme-muted"
                      />
                    </div>
                    <div className="text-xs text-theme-secondary">
                      当前命中 {filteredGroups.length} /{" "}
                      {watchlistGroups.length} 个分组
                    </div>
                    <div className="text-xs text-theme-muted">
                      {reorderEnabled
                        ? "当前可直接拖拽下方分组卡片调整顺序。"
                        : normalizedGroupSearch
                          ? "如需拖拽排序，请先清空搜索关键词。"
                          : groupViewMode === "focused"
                            ? "如需拖拽排序，请切回“全部分组”模式。"
                            : "至少需要两个分组后才可拖拽排序。"}
                    </div>
                  </div>

                  <div className="space-y-3">
                    <div className="text-sm text-theme-muted">浏览模式</div>
                    <div className="grid gap-3 sm:grid-cols-2">
                      {(
                        [
                          {
                            id: "all",
                            label: "全部分组",
                            icon: Layers3,
                            eyebrow: "浏览全部",
                            description: "完整浏览全部分组",
                            helper: `当前高亮：${focusedGroup?.name ?? "未选中"}`,
                          },
                          {
                            id: "focused",
                            label: "当前分组",
                            icon: Check,
                            eyebrow: "聚焦查看",
                            description: "只看当前浏览分组",
                            helper: focusedGroup
                              ? `当前聚焦：${focusedGroup.name}`
                              : "先选择一个分组",
                          },
                        ] as const
                      ).map((option) => {
                        const Icon = option.icon;
                        const active = groupViewMode === option.id;
                        return (
                          <button
                            key={option.id}
                            type="button"
                            onClick={() => handleChangeGroupViewMode(option.id)}
                            className={cn(
                              "group relative overflow-hidden rounded-[26px] border px-4 py-4 text-left transition-all duration-200",
                              "hover:-translate-y-0.5 active:scale-[0.985]",
                              active
                                ? "border-cyan-400/45 bg-gradient-to-br from-cyan-500/16 via-sky-500/10 to-transparent text-theme-primary shadow-[0_16px_34px_rgba(34,211,238,0.16)]"
                                : "border-[var(--input-border)] bg-[var(--input-bg)]/70 text-theme-secondary hover:border-cyan-400/35 hover:bg-cyan-500/8 hover:text-theme-primary hover:shadow-[0_12px_26px_rgba(34,211,238,0.10)]",
                              animatingViewMode === option.id &&
                                "action-button-pop",
                            )}
                          >
                            <span className="action-button-shine" />
                            <div className="relative z-10 flex items-start justify-between gap-3">
                              <div className="min-w-0">
                                <div className="text-[11px] tracking-[0.24em] text-theme-muted">
                                  {option.eyebrow}
                                </div>
                                <div className="mt-2 flex items-center gap-2">
                                  <Icon
                                    className={cn(
                                      "h-4 w-4 transition-transform duration-300",
                                      active
                                        ? "scale-110 text-cyan-200"
                                        : "group-hover:scale-110 group-hover:text-cyan-200",
                                    )}
                                  />
                                  <span className="text-sm font-semibold text-theme-primary">
                                    {option.label}
                                  </span>
                                </div>
                              </div>
                              <span
                                className={cn(
                                  "rounded-full border px-2 py-1 text-[10px] tracking-[0.18em] transition-all duration-300",
                                  active
                                    ? "border-cyan-400/35 bg-cyan-400/12 text-cyan-200"
                                    : "border-[var(--input-border)] bg-[var(--card-bg)]/50 text-theme-muted",
                                )}
                              >
                                {active ? "当前" : "切换"}
                              </span>
                            </div>
                            <div className="relative z-10 mt-4 space-y-1">
                              <div className="text-sm font-medium text-theme-primary">
                                {option.description}
                              </div>
                              <div className="text-xs leading-5 text-theme-secondary">
                                {option.helper}
                              </div>
                            </div>
                          </button>
                        );
                      })}
                    </div>
                    <div className="text-xs text-theme-secondary">
                      {groupViewMode === "focused"
                        ? `当前只展示分组「${focusedGroup?.name ?? "未选中"}」，适合分组很多时快速聚焦`
                        : `当前展示全部分组，正在高亮「${focusedGroup?.name ?? "未选中"}」并支持快速跳转`}
                    </div>
                  </div>
                </div>

                {focusedGroup && (
                  <div className="mt-5 rounded-[24px] border border-cyan-500/25 bg-cyan-500/10 px-4 py-4 shadow-[0_16px_30px_rgba(34,211,238,0.10)]">
                    <div className="text-[11px] tracking-[0.24em] text-cyan-300">
                      当前浏览分组
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <span className="rounded-full border border-cyan-400/35 bg-cyan-400/10 px-2.5 py-1 text-[11px] tracking-[0.18em] text-cyan-200">
                        {groupViewMode === "focused"
                          ? "当前仅展示"
                          : "当前高亮"}
                      </span>
                      <span className="text-lg font-bold text-theme-primary sm:text-xl">
                        {focusedGroup.name}
                      </span>
                    </div>
                    <div className="mt-2 text-sm leading-6 text-theme-secondary">
                      {groupViewMode === "focused"
                        ? `下方当前只显示「${focusedGroup.name}」一个分组。切回“全部分组”后，可继续浏览其它分组。`
                        : `下方完整列表里会重点展示「${focusedGroup.name}」，点击分组导航可快速切换当前浏览分组。`}
                    </div>
                  </div>
                )}

                <div className="mt-5">
                  <div className="mb-3 text-sm text-theme-muted">分组导航</div>
                  {filteredGroups.length === 0 ? (
                    <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-6 text-sm text-theme-secondary">
                      没有匹配的分组，试试别的关键词。
                    </div>
                  ) : (
                    <div className="flex gap-2 overflow-x-auto pb-2">
                      {filteredGroups.map((group) => {
                        const active = group.id === focusedGroupId;
                        return (
                          <button
                            key={group.id}
                            type="button"
                            onClick={() => handleFocusGroup(group.id)}
                            className={cn(
                              "group relative shrink-0 overflow-hidden rounded-full border px-3 py-2 text-xs transition-all duration-200",
                              "hover:-translate-y-0.5 hover:shadow-[0_10px_22px_rgba(34,211,238,0.10)] active:scale-[0.97]",
                              active
                                ? "border-cyan-400/50 bg-cyan-400/15 text-cyan-100 shadow-[0_10px_22px_rgba(34,211,238,0.12)]"
                                : "border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary",
                              animatingNavGroupId === group.id &&
                                "action-button-pop",
                            )}
                          >
                            <span className="action-button-shine" />
                            <span className="relative z-10 inline-flex items-center gap-1.5">
                              {active && (
                                <span className="h-1.5 w-1.5 rounded-full bg-cyan-200 shadow-[0_0_10px_rgba(165,243,252,0.9)]" />
                              )}
                              {group.name}
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  )}
                </div>
              </Surface>

              {visibleGroups.length === 0 ? (
                <EmptyState
                  icon={<Layers3 className="h-10 w-10" />}
                  title="当前没有可展示的分组"
                  description="请检查搜索关键词，或切回“全部分组”模式查看完整列表。"
                />
              ) : (
                visibleGroups.map((group) => {
                  const isCollapsed = collapsedGroupIds.has(group.id);
                  const isFocused = group.id === focusedGroupId;
                  const isDragging = draggingGroupId === group.id;
                  const isDragOver =
                    dragOverGroupId === group.id &&
                    draggingGroupId !== group.id;
                  return (
                    <section
                      key={group.id}
                      ref={(node) => {
                        groupSectionRefs.current[group.id] = node;
                      }}
                      onDragOver={(event) =>
                        handleGroupDragOver(event, group.id)
                      }
                      onDrop={() => void handleGroupDrop(group.id)}
                      onDragEnd={() => {
                        setDraggingGroupId(null);
                        setDragOverGroupId(null);
                      }}
                      id={`watchlist-group-${group.id}`}
                      style={{
                        scrollMarginTop:
                          "var(--account-shell-anchor-offset, 112px)",
                      }}
                      className={cn(
                        "rounded-[32px] border border-[var(--card-border)] p-6 glass transition-all duration-200",
                        isDragging &&
                          "scale-[0.99] border-cyan-400/35 shadow-[0_20px_40px_rgba(34,211,238,0.12)] opacity-80",
                        isDragOver &&
                          "border-cyan-400/40 shadow-[0_18px_34px_rgba(34,211,238,0.10)]",
                      )}
                    >
                      <div
                        className={`mb-6 rounded-[28px] bg-gradient-to-r ${watchlistAccentToClass(group.accent)} p-5`}
                      >
                        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                          <div>
                            <div className="flex flex-wrap items-center gap-2">
                              {reorderEnabled && (
                                <button
                                  type="button"
                                  draggable
                                  onDragStart={(event) =>
                                    handleGroupDragStart(event, group.id)
                                  }
                                  onDragEnd={() => {
                                    setDraggingGroupId(null);
                                    setDragOverGroupId(null);
                                  }}
                                  className={cn(
                                    "group/drag inline-flex cursor-grab select-none items-center gap-1 rounded-full border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-2 py-1 text-[11px] text-theme-muted transition-all duration-200 active:cursor-grabbing",
                                    "hover:border-cyan-400/35 hover:bg-cyan-400/10 hover:text-theme-primary",
                                    isDragging &&
                                      "border-cyan-400/45 bg-cyan-400/14 text-cyan-100 action-button-pop",
                                  )}
                                  aria-label={`拖拽调整 ${group.name} 的排序`}
                                  title="拖拽调整排序"
                                >
                                  <GripVertical className="h-3.5 w-3.5 transition-transform duration-200 group-hover/drag:scale-110" />
                                  拖拽排序
                                </button>
                              )}
                              <div className="text-2xl font-black text-theme-primary">
                                {group.name}
                              </div>
                              {isFocused && (
                                <span className="rounded-full border border-cyan-400/35 bg-cyan-400/10 px-2 py-1 text-[11px] tracking-[0.18em] text-cyan-200">
                                  当前分组
                                </span>
                              )}
                              <span
                                className={cn(
                                  "rounded-full border px-2 py-1 text-[11px]",
                                  watchlistAccentBadgeClass(group.accent),
                                )}
                              >
                                {watchlistAccentLabel(group.accent)}
                              </span>
                            </div>
                            <p className="mt-2 max-w-2xl text-sm leading-6 text-theme-secondary">
                              {group.description || "未填写分组说明"}
                            </p>
                            <div className="mt-3 text-xs text-theme-muted">
                              共 {group.funds.length} 只基金
                            </div>
                          </div>

                          <div className="flex flex-wrap items-center gap-3">
                            <button
                              type="button"
                              draggable={false}
                              onMouseDown={(event) => event.stopPropagation()}
                              onDragStart={(event) => event.preventDefault()}
                              onClick={(event) => {
                                event.stopPropagation();
                                openGroupEditor(group);
                              }}
                              disabled={
                                deletingGroupID !== null ||
                                savingGroupId === group.id
                              }
                              className={cn(
                                "group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-all duration-200",
                                "hover:-translate-y-0.5 hover:border-cyan-400/35 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)] active:scale-[0.985]",
                                "disabled:cursor-not-allowed disabled:opacity-70",
                                animatingEditGroupId === group.id &&
                                  "action-button-pop border-cyan-400/40 bg-cyan-500/12 text-theme-primary",
                              )}
                            >
                              <span className="action-button-shine" />
                              <PencilLine className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110" />
                              <span className="relative z-10">编辑分组</span>
                            </button>
                            <button
                              type="button"
                              onClick={() => toggleGroupCollapse(group.id)}
                              className={cn(
                                "group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-all duration-200",
                                "hover:-translate-y-0.5 hover:border-cyan-400/35 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)] active:scale-[0.985]",
                                animatingCollapseGroupId === group.id &&
                                  "action-button-pop border-cyan-400/40 bg-cyan-500/12 text-theme-primary",
                              )}
                            >
                              <span className="action-button-shine" />
                              <ChevronDown
                                className={cn(
                                  "relative z-10 h-4 w-4 transition-transform duration-300",
                                  isCollapsed ? "rotate-0" : "rotate-180",
                                  animatingCollapseGroupId === group.id &&
                                    "scale-110",
                                )}
                              />
                              <span className="relative z-10">
                                {isCollapsed ? "展开分组" : "收起分组"}
                              </span>
                            </button>

                            <button
                              type="button"
                              onClick={() => void handleDeleteGroup(group.id)}
                              disabled={
                                deletingGroupID !== null ||
                                savingGroupId === group.id
                              }
                              className={cn(
                                "group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-all duration-200",
                                "hover:-translate-y-0.5 hover:border-rose-400/40 hover:bg-rose-500/12 hover:text-rose-200 active:scale-[0.985]",
                                "disabled:cursor-not-allowed disabled:opacity-80",
                                deletingGroupID === group.id &&
                                  "danger-button-pop border-rose-400/45 bg-rose-500/14 text-rose-100",
                              )}
                            >
                              <span className="action-button-shine" />
                              {deletingGroupID === group.id ? (
                                <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
                              ) : (
                                <Trash2 className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-12 group-hover:scale-110" />
                              )}
                              <span className="relative z-10">
                                {deletingGroupID === group.id
                                  ? "删除中..."
                                  : "删除分组"}
                              </span>
                            </button>
                          </div>
                        </div>
                      </div>

                      {isCollapsed ? (
                        <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-5 py-6 text-center text-sm text-theme-secondary">
                          当前分组已折叠，展开后可查看分组内基金。
                        </div>
                      ) : group.funds.length === 0 ? (
                        <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-5 py-10 text-center text-sm text-theme-secondary">
                          当前分组还没有基金，从上面的搜索结果里把基金加入这里。
                        </div>
                      ) : (
                        <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
                          {group.funds.map((item, index) => (
                            <ScrollReveal
                              key={`${group.id}:${item.fund_id}`}
                              delay={Math.min(index * 70, 280)}
                              className="h-full"
                            >
                              <WatchlistFundCard
                                fundId={item.fund_id}
                                analysis={analysesByFundID[item.fund_id]}
                                history={historiesByFundID[item.fund_id]}
                                isHistoryLoading={isHistoryLoading}
                                onRemove={() =>
                                  void removeFundFromGroup(
                                    group.id,
                                    item.fund_id,
                                  )
                                }
                              />
                            </ScrollReveal>
                          ))}
                        </div>
                      )}
                    </section>
                  );
                })
              )}
            </>
          )}
        </div>

        {editingGroupId && (
          <>
            <div
              className="watchlist-editor-backdrop fixed inset-0 z-[70] bg-slate-950/62 backdrop-blur-sm"
              onClick={closeGroupEditor}
            />
            <div className="pointer-events-none fixed inset-0 z-[71] flex items-center justify-center px-4 py-6">
              <div className="watchlist-editor-panel pointer-events-auto w-full max-w-2xl rounded-[32px] border border-[var(--card-border)] bg-[var(--card-bg)]/95 p-6 shadow-[0_28px_80px_rgba(2,8,23,0.40)] backdrop-blur-xl">
                <div className="mb-6 flex items-start justify-between gap-4">
                  <div>
                    <div className="text-xs tracking-[0.24em] text-cyan-300">
                      编辑分组
                    </div>
                    <div className="mt-2 text-2xl font-bold text-theme-primary">
                      修改分组信息
                    </div>
                    <div className="mt-2 text-sm leading-6 text-theme-secondary">
                      可同时调整分组名称、说明和颜色标识，保存后会立即同步到分组列表和导航区域。
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={closeGroupEditor}
                    className="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary transition-all duration-200 hover:border-cyan-400/35 hover:text-theme-primary"
                    aria-label="关闭编辑分组弹窗"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>

                <div className="space-y-5">
                  <label className="block">
                    <div className="mb-2 text-sm text-theme-secondary">
                      分组名称
                    </div>
                    <input
                      value={editingGroupName}
                      onChange={(event) =>
                        setEditingGroupName(event.target.value)
                      }
                      placeholder="例如：核心观察"
                      className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-base font-semibold text-theme-primary outline-none placeholder:text-theme-muted"
                    />
                  </label>

                  <label className="block">
                    <div className="mb-2 text-sm text-theme-secondary">
                      分组说明
                    </div>
                    <textarea
                      value={editingGroupDescription}
                      onChange={(event) =>
                        setEditingGroupDescription(event.target.value)
                      }
                      rows={4}
                      placeholder="补充一句说明，方便区分这个分组的用途和关注重点"
                      className="auth-input w-full resize-y rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm leading-6 text-theme-primary outline-none placeholder:text-theme-muted"
                    />
                  </label>

                  <div>
                    <div className="mb-3 flex items-center gap-2 text-sm text-theme-secondary">
                      <Palette className="h-4 w-4 text-cyan-300" />
                      分组颜色
                    </div>
                    <div className="grid gap-3 sm:grid-cols-2">
                      {GROUP_ACCENT_OPTIONS.map((option) => {
                        const active = editingGroupAccent === option.value;
                        return (
                          <button
                            key={option.value}
                            type="button"
                            onClick={() => setEditingGroupAccent(option.value)}
                            className={cn(
                              "group rounded-[22px] border px-4 py-4 text-left transition-all duration-200",
                              active
                                ? `${option.shell} shadow-[0_14px_28px_rgba(15,23,42,0.12)]`
                                : "border-[var(--input-border)] bg-[var(--input-bg)]/70 text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary",
                            )}
                          >
                            <div className="flex items-center justify-between gap-3">
                              <div className="flex items-center gap-3">
                                <span
                                  className={cn(
                                    "h-3 w-3 rounded-full shadow-[0_0_12px_rgba(255,255,255,0.18)]",
                                    option.dot,
                                  )}
                                />
                                <span className="text-sm font-medium">
                                  {option.label}
                                </span>
                              </div>
                              {active && (
                                <Check className="h-4 w-4 text-cyan-100" />
                              )}
                            </div>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </div>

                <div className="mt-6 flex flex-wrap justify-end gap-3">
                  <button
                    type="button"
                    onClick={closeGroupEditor}
                    disabled={savingGroupId !== null}
                    className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-5 py-3 text-sm text-theme-secondary transition-all duration-200 hover:border-cyan-400/35 hover:text-theme-primary disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    取消
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      const currentGroup = orderedWatchlistGroups.find(
                        (group) => group.id === editingGroupId,
                      );
                      if (currentGroup) {
                        void handleUpdateGroup(currentGroup);
                      }
                    }}
                    disabled={savingGroupId !== null}
                    className={cn(
                      "group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-5 py-3 text-sm font-medium text-white transition-all duration-200",
                      "hover:-translate-y-0.5 hover:shadow-[0_18px_35px_rgba(14,165,233,0.28)] active:scale-[0.985]",
                      "disabled:cursor-not-allowed disabled:opacity-80",
                      savingGroupId !== null && "action-button-pop",
                    )}
                  >
                    <span className="action-button-shine" />
                    {savingGroupId !== null ? (
                      <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
                    ) : (
                      <Check className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:scale-110" />
                    )}
                    <span className="relative z-10">
                      {savingGroupId !== null ? "保存中..." : "保存修改"}
                    </span>
                  </button>
                </div>
              </div>
            </div>
          </>
        )}
      </ScrollRevealStack>
    </AccountAreaShell>
  );
}
