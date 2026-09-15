"use client";

import React, { useEffect, useState, useRef } from "react";

type Priority = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

interface RankedEmail {
  gmail_id: string;
  thread_id: string;
  sender: string;
  recipient: string;
  subject: string;
  received_at: string;
  snippet: string;
  body: string;
  category: string;
  priority: Priority;
  summary: string;
  action_required: boolean;
  deadline: string | null;
  attention_score: number;
  ai_processed_at: string | null;
  is_read: boolean;
  is_replied: boolean;
  is_archived: boolean;
  is_starred: boolean;
  read_at: string | null;
  replied_at: string | null;
  draft_reply: string;
}

interface EmailStats {
  total: number;
  last_12h: number;
  last_24h: number;
  last_7d: number;
  last_30d: number;
  unread: number;
  read: number;
  replied: number;
  starred: number;
  action_required: number;
  categories: Record<string, number>;
}

const categoryConfig: Record<string, { label: string; icon: string; bg: string; text: string; border: string }> = {
  LEO: { label: "Leo Club", icon: "🦁", bg: "bg-amber-500/15", text: "text-amber-400", border: "border-amber-500/30" },
  IEEE: { label: "IEEE Branch", icon: "⚡", bg: "bg-cyan-500/15", text: "text-cyan-400", border: "border-cyan-500/30" },
  UNI: { label: "Faculty / Uni", icon: "🎓", bg: "bg-purple-500/15", text: "text-purple-400", border: "border-purple-500/30" },
  JOB: { label: "Jobs & Careers", icon: "💼", bg: "bg-emerald-500/15", text: "text-emerald-400", border: "border-emerald-500/30" },
  SECURITY: { label: "Security & 2FA", icon: "🔒", bg: "bg-rose-500/15", text: "text-rose-400", border: "border-rose-500/30" },
  FINANCE: { label: "Finance & Bills", icon: "💰", bg: "bg-teal-500/15", text: "text-teal-400", border: "border-teal-500/30" },
  WORK: { label: "Work & Tasks", icon: "💻", bg: "bg-blue-500/15", text: "text-blue-400", border: "border-blue-500/30" },
  PERSONAL: { label: "Personal", icon: "👤", bg: "bg-indigo-500/15", text: "text-indigo-400", border: "border-indigo-500/30" },
  PROMOTION: { label: "Promotions", icon: "🎁", bg: "bg-pink-500/15", text: "text-pink-400", border: "border-pink-500/30" },
  SOCIAL: { label: "Social", icon: "👥", bg: "bg-sky-500/15", text: "text-sky-400", border: "border-sky-500/30" },
  OTHER: { label: "General", icon: "📁", bg: "bg-zinc-500/15", text: "text-zinc-400", border: "border-zinc-500/30" },
};

const priorityConfig: Record<string, { dot: string; text: string; badge: string }> = {
  CRITICAL: { dot: "bg-rose-500 shadow-[0_0_8px_rgba(244,63,94,0.6)]", text: "text-rose-400", badge: "bg-rose-500/20 text-rose-300 border-rose-500/30" },
  HIGH: { dot: "bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.6)]", text: "text-amber-400", badge: "bg-amber-500/20 text-amber-300 border-amber-500/30" },
  MEDIUM: { dot: "bg-yellow-500", text: "text-yellow-400", badge: "bg-yellow-500/15 text-yellow-300 border-yellow-500/30" },
  LOW: { dot: "bg-emerald-500", text: "text-emerald-400", badge: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30" },
};

function getInitials(name: string): string {
  if (!name) return "M";
  const clean = name.replace(/<.*>/, "").trim();
  const parts = clean.split(" ").filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return clean.slice(0, 2).toUpperCase();
}

function getCleanSenderName(sender: string): string {
  if (!sender) return "Unknown Sender";
  const match = sender.match(/^([^<]+)/);
  if (match && match[1].trim()) {
    return match[1].trim().replace(/^"|"$/g, "");
  }
  return sender;
}

function getSenderEmail(sender: string): string {
  if (!sender) return "";
  const match = sender.match(/<([^>]+)>/);
  return match ? match[1] : sender;
}

function formatShortDate(dateString: string): string {
  if (!dateString) return "";
  const date = new Date(dateString);
  const now = new Date();
  const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (diffInSeconds < 60) return "Just now";
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
  if (diffInSeconds < 172800) return "Yesterday";
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function formatRelativeDeadline(deadlineString: string | null): { text: string; urgent: boolean } | null {
  if (!deadlineString) return null;
  const deadline = new Date(deadlineString);
  const now = new Date();
  const diffHours = (deadline.getTime() - now.getTime()) / (1000 * 60 * 60);

  if (diffHours < 0) return { text: "Overdue", urgent: true };
  if (diffHours <= 24) return { text: `Due in ${Math.round(diffHours)}h`, urgent: true };
  if (diffHours <= 72) return { text: `Due in ${Math.round(diffHours / 24)}d`, urgent: false };
  return { text: `Due ${deadline.toLocaleDateString(undefined, { month: "short", day: "numeric" })}`, urgent: false };
}

export default function Home() {
  const [emails, setEmails] = useState<RankedEmail[]>([]);
  const [stats, setStats] = useState<EmailStats | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // Client-side In-Memory SWR Cache
  const cacheRef = useRef<Record<string, RankedEmail[]>>({});

  // Filters state
  const [timeframe, setTimeframe] = useState<string>("all"); // "12h", "24h", "7d", "30d", "all"
  const [statusFilter, setStatusFilter] = useState<string>("all"); // "all", "unread", "read", "replied", "starred", "action"
  const [activeCategory, setActiveCategory] = useState<string>("ALL");
  const [searchQuery, setSearchQuery] = useState("");

  // Responsive & Mobile State
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);
  const [mobileView, setMobileView] = useState<"list" | "reader">("list"); // for screens < md

  // App UI State
  const [loading, setLoading] = useState(true);
  const [isSyncing, setIsSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  useEffect(() => {
    setMounted(true);
  }, []);

  const fetchStats = async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/emails/stats`);
      if (res.ok) {
        const data: EmailStats = await res.json();
        setStats(data);
      }
    } catch (e) {
      console.warn("Failed to fetch stats:", e);
    }
  };

  const fetchEmails = async (triggerSync = false) => {
    const cacheKey = `${timeframe}:${statusFilter}:${activeCategory}:${searchQuery.trim()}`;

    // Instant UI Cache Hit
    if (!triggerSync && cacheRef.current[cacheKey]) {
      setEmails(cacheRef.current[cacheKey]);
      setLoading(false);
    } else if (emails.length === 0) {
      setLoading(true);
    }

    try {
      if (triggerSync) {
        setIsSyncing(true);
        try {
          await fetch(`${API_BASE_URL}/api/sync`, { method: "POST" });
        } catch (e) {
          console.warn("Sync trigger warning:", e);
        }
      }

      setError(null);

      const params = new URLSearchParams();
      if (timeframe && timeframe !== "all") params.append("timeframe", timeframe);
      if (statusFilter && statusFilter !== "all") params.append("status", statusFilter);
      if (activeCategory && activeCategory !== "ALL") params.append("category", activeCategory);
      if (searchQuery.trim()) params.append("search", searchQuery.trim());
      params.append("limit", "100");

      const res = await fetch(`${API_BASE_URL}/api/emails/priority?${params.toString()}`);
      if (!res.ok) {
        throw new Error(`API error (${res.status})`);
      }
      const data: RankedEmail[] = await res.json();
      
      // Update state and cache
      cacheRef.current[cacheKey] = data || [];
      setEmails(data || []);

      if (data && data.length > 0) {
        if (!selectedId || !data.some((e) => e.gmail_id === selectedId)) {
          setSelectedId(data[0].gmail_id);
        }
      }

      fetchStats();
    } catch (err: any) {
      console.error("Failed to fetch emails:", err);
      setError(`Unable to connect to API server (${API_BASE_URL})`);
    } finally {
      setLoading(false);
      setIsSyncing(false);
    }
  };

  useEffect(() => {
    fetchEmails(false);
  }, [timeframe, statusFilter, activeCategory]);

  useEffect(() => {
    fetchEmails(true);
  }, []);

  const handleUpdateStatus = async (gmailId: string, updates: Partial<RankedEmail>) => {
    setEmails((prev) =>
      prev.map((e) => (e.gmail_id === gmailId ? { ...e, ...updates } : e))
    );

    for (const key in cacheRef.current) {
      cacheRef.current[key] = cacheRef.current[key].map((e) =>
        e.gmail_id === gmailId ? { ...e, ...updates } : e
      );
    }

    try {
      await fetch(`${API_BASE_URL}/api/emails/status`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          gmail_id: gmailId,
          ...updates,
        }),
      });
      fetchStats();
    } catch (err) {
      console.error("Failed to update email status:", err);
    }
  };

  const selectedEmail = emails.find((e) => e.gmail_id === selectedId) || emails[0] || null;

  useEffect(() => {
    if (selectedEmail && !selectedEmail.is_read) {
      handleUpdateStatus(selectedEmail.gmail_id, { is_read: true });
    }
  }, [selectedId]);

  // Keyboard shortcut listener for Search (⌘K / Ctrl+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        const searchInput = document.getElementById("search-input");
        searchInput?.focus();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  // Shared Sidebar Component Content
  const renderSidebarContent = () => (
    <div className="flex h-full flex-col p-3 text-xs select-none">
      {/* Brand & Window Controls */}
      <div className="mb-4 flex items-center justify-between px-2 pt-1">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-tr from-amber-500 via-orange-500 to-rose-500 font-black text-white shadow-md text-sm">
            M
          </div>
          <span className="font-bold text-[14px] tracking-tight text-white">MailMind</span>
          <span className="rounded bg-[#1e222d] px-1.5 py-0.5 text-[9px] font-mono text-amber-400 font-semibold border border-[#2b303f]">
            AI Triage
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="h-2.5 w-2.5 rounded-full bg-[#ff5f57]" />
          <div className="h-2.5 w-2.5 rounded-full bg-[#febc2e]" />
          <div className="h-2.5 w-2.5 rounded-full bg-[#28c840]" />
        </div>
      </div>

      {/* User Card */}
      <div className="mb-4 flex items-center gap-2.5 rounded-xl border border-[#1e222d] bg-[#14171f]/80 p-2 shadow-sm">
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-amber-600 to-orange-500 font-bold text-white text-xs shadow-inner">
          MR
        </div>
        <div className="min-w-0 flex-1">
          <div className="truncate font-semibold text-white text-[12px]">Mesith Rathnayake</div>
          <div className="truncate text-[10px] text-[#717684]">
            {emails.find((e) => e.recipient)?.recipient || "Gmail Sync Active"}
          </div>
        </div>
      </div>

      {/* Search Input */}
      <div className="relative mb-4">
        <span className="absolute left-2.5 top-2 text-[11px] text-[#717684]">🔍</span>
        <input
          id="search-input"
          type="text"
          placeholder="Search emails (⌘K)..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") fetchEmails(false);
          }}
          className="h-8 w-full rounded-lg border border-[#1e222d] bg-[#14171f] pl-7 pr-7 text-xs text-white placeholder-[#5a5f6e] focus:outline-none focus:border-amber-500/50 focus:ring-1 focus:ring-amber-500/50 transition"
        />
        {searchQuery && (
          <button
            onClick={() => {
              setSearchQuery("");
              setTimeout(() => fetchEmails(false), 0);
            }}
            className="absolute right-2 top-2 text-[11px] text-[#717684] hover:text-white"
          >
            ✕
          </button>
        )}
      </div>

      {/* Views & Status */}
      <div className="mb-2 px-1 text-[10px] font-bold uppercase tracking-wider text-[#535868]">
        Views & Status
      </div>
      <div className="space-y-0.5">
        {[
          { key: "all", name: "All Priority", icon: "⭐", count: stats?.total },
          { key: "unread", name: "Unread / Unopened", icon: "🔵", count: stats?.unread, badgeColor: "bg-blue-500/20 text-blue-300 font-bold border border-blue-500/30" },
          { key: "read", name: "Read / Viewed", icon: "👁️", count: stats?.read },
          { key: "action", name: "Action Required", icon: "⚡", count: stats?.action_required, badgeColor: "bg-amber-500/20 text-amber-300 font-bold border border-amber-500/30" },
          { key: "replied", name: "Replied", icon: "↩️", count: stats?.replied },
          { key: "starred", name: "Favorites", icon: "📌", count: stats?.starred },
        ].map((item) => (
          <button
            key={item.key}
            onClick={() => {
              setStatusFilter(item.key);
              setIsMobileSidebarOpen(false);
            }}
            className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 transition ${
              statusFilter === item.key
                ? "bg-[#1c202a] font-semibold text-white shadow-sm border border-[#2b303f]"
                : "text-[#8e93a2] hover:bg-[#15171e] hover:text-white"
            }`}
          >
            <div className="flex items-center gap-2">
              <span className="text-[12px]">{item.icon}</span>
              <span className="text-[12px]">{item.name}</span>
            </div>
            {item.count !== undefined && item.count > 0 && (
              <span
                className={`rounded-full px-2 py-0.2 text-[10px] ${
                  item.badgeColor || (statusFilter === item.key ? "bg-[#2c303f] text-white" : "text-[#717684]")
                }`}
              >
                {item.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Hubs / Categories */}
      <div className="mt-4 mb-2 flex items-center justify-between px-1 text-[10px] font-bold uppercase tracking-wider text-[#535868]">
        <span>Category Hubs</span>
        <span className="text-xs">▾</span>
      </div>
      <div className="flex-1 overflow-y-auto space-y-0.5 no-scrollbar pr-1">
        <button
          onClick={() => {
            setActiveCategory("ALL");
            setIsMobileSidebarOpen(false);
          }}
          className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 transition text-[12px] ${
            activeCategory === "ALL"
              ? "bg-[#1c202a] font-semibold text-white border border-[#2b303f]"
              : "text-[#8e93a2] hover:bg-[#15171e] hover:text-white"
          }`}
        >
          <div className="flex items-center gap-2">
            <span>🌐</span>
            <span>All Categories</span>
          </div>
          {stats?.total !== undefined && (
            <span className="text-[10px] text-[#717684]">{stats.total}</span>
          )}
        </button>

        {Object.entries(categoryConfig).map(([key, config]) => {
          const count = stats?.categories?.[key] || 0;
          return (
            <button
              key={key}
              onClick={() => {
                setActiveCategory(key);
                setIsMobileSidebarOpen(false);
              }}
              className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 transition text-[12px] ${
                activeCategory === key
                  ? "bg-[#1c202a] font-semibold text-white border border-[#2b303f]"
                  : "text-[#8e93a2] hover:bg-[#15171e] hover:text-white"
              }`}
            >
              <div className="flex items-center gap-2">
                <span>{config.icon}</span>
                <span>{config.label}</span>
              </div>
              {count > 0 && (
                <span className={`rounded-full px-1.5 text-[10px] font-semibold ${config.bg} ${config.text}`}>
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Sync Footer */}
      <div className="mt-auto border-t border-[#1a1c23] pt-3">
        <button
          onClick={() => fetchEmails(true)}
          disabled={isSyncing}
          className="flex w-full items-center justify-between rounded-lg border border-[#1e222d] bg-[#14171f] px-3 py-2 text-[#8e93a2] hover:bg-[#1c202a] hover:text-white transition disabled:opacity-50"
        >
          <div className="flex items-center gap-2">
            <span className={`h-2 w-2 rounded-full ${isSyncing ? "bg-amber-400 animate-ping" : error ? "bg-red-500" : "bg-emerald-400"}`} />
            <span className="font-medium text-[11px]">{isSyncing ? "Syncing Gmail & AI..." : "Live Pipeline Sync"}</span>
          </div>
          <span className={`text-xs ${isSyncing ? "animate-spin text-amber-400" : ""}`}>↻</span>
        </button>
      </div>
    </div>
  );

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#090a0d] font-sans text-[#e2e4e9] antialiased selection:bg-[#32363f]">
      {/* ========================================================================= */}
      {/* MOBILE DRAWER BACKDROP & OVERLAY */}
      {/* ========================================================================= */}
      {isMobileSidebarOpen && (
        <div
          onClick={() => setIsMobileSidebarOpen(false)}
          className="fixed inset-0 z-40 bg-black/70 backdrop-blur-sm md:hidden transition-opacity"
        />
      )}

      {/* MOBILE DRAWER SIDEBAR */}
      <div
        className={`fixed inset-y-0 left-0 z-50 w-72 bg-[#0e1014] border-r border-[#1a1c23] transform transition-transform duration-300 ease-in-out md:hidden ${
          isMobileSidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        {renderSidebarContent()}
      </div>

      {/* ========================================================================= */}
      {/* COLUMN 1: DESKTOP LEFT SIDEBAR */}
      {/* ========================================================================= */}
      <aside className="hidden md:flex w-60 lg:w-64 shrink-0 flex-col border-r border-[#1a1c23] bg-[#0e1014]">
        {renderSidebarContent()}
      </aside>

      {/* ========================================================================= */}
      {/* COLUMN 2: EMAIL LIST STREAM (Middle Pane) */}
      {/* ========================================================================= */}
      <section
        className={`flex flex-col border-r border-[#1a1c23] bg-[#0c0d11] shrink-0 ${
          mobileView === "reader"
            ? "hidden md:flex md:w-80 lg:w-[380px] xl:w-[410px]"
            : "w-full md:w-80 lg:w-[380px] xl:w-[410px]"
        }`}
      >
        {/* Header with Title, Mobile Drawer Trigger, & Sync status */}
        <div className="flex items-center justify-between border-b border-[#1a1c23] px-3.5 py-3">
          <div className="flex items-center gap-2">
            {/* Mobile Hamburger Button */}
            <button
              onClick={() => setIsMobileSidebarOpen(true)}
              className="flex h-8 w-8 items-center justify-center rounded-lg border border-[#232733] bg-[#14171f] text-[#8e93a2] hover:text-white md:hidden"
              title="Open Navigation"
            >
              ☰
            </button>
            <h2 className="text-[14px] font-bold text-white flex items-center gap-1.5 truncate">
              <span>{activeCategory !== "ALL" ? categoryConfig[activeCategory]?.label || activeCategory : "Priority Inbox"}</span>
              <span className="text-[11px] font-normal text-[#717684]">({emails.length})</span>
            </h2>
          </div>

          <button
            onClick={() => fetchEmails(true)}
            disabled={isSyncing}
            className="flex items-center gap-1.5 rounded-lg border border-[#232733] bg-[#14171f] px-2.5 py-1 text-[11px] font-medium text-[#8e93a2] hover:text-white hover:border-[#373c4d] transition disabled:opacity-50 shrink-0"
            title="Sync Latest Emails & AI Triage"
          >
            <span className={isSyncing ? "animate-spin text-amber-400" : ""}>↻</span>
            <span className="hidden sm:inline">{isSyncing ? "Syncing..." : "Sync"}</span>
          </button>
        </div>

        {/* TIME FRAMING SELECTOR PILLS */}
        <div className="border-b border-[#1a1c23] bg-[#101217] px-3 py-2">
          <div className="mb-1.5 flex items-center justify-between">
            <span className="text-[10px] font-bold uppercase tracking-wider text-[#636878]">
              ⏰ Time Frame Filter
            </span>
            {stats && (
              <span className="text-[10px] text-amber-400 font-mono">
                {timeframe === "12h" ? `${stats.last_12h} while sleeping` : timeframe === "24h" ? `${stats.last_24h} today` : ""}
              </span>
            )}
          </div>
          <div className="grid grid-cols-5 gap-1 text-center">
            {[
              { key: "12h", label: "🌙 12h", title: "Last 12 hours (While sleeping)" },
              { key: "24h", label: "☀️ 24h", title: "Last 24 hours (Today)" },
              { key: "7d", label: "📅 7d", title: "Last 7 days (This week)" },
              { key: "30d", label: "🗓️ 30d", title: "Last 30 days (This month)" },
              { key: "all", label: "🌐 All", title: "All time" },
            ].map((t) => (
              <button
                key={t.key}
                onClick={() => setTimeframe(t.key)}
                title={t.title}
                className={`rounded-lg py-1 text-[11px] font-semibold transition truncate ${
                  timeframe === t.key
                    ? "bg-gradient-to-r from-amber-500/20 to-orange-500/20 text-amber-300 border border-amber-500/40 shadow-sm"
                    : "bg-[#151820] text-[#717684] hover:bg-[#1a1e28] hover:text-white border border-transparent"
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* Emails Stream List */}
        <div className="flex-1 overflow-y-auto px-2 py-2 space-y-1.5">
          {loading && emails.length === 0 ? (
            <div className="p-8 text-center text-xs text-[#717684]">
              <div className="mb-2 animate-spin text-lg text-amber-400">↻</div>
              Loading &amp; ranking emails...
            </div>
          ) : emails.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-8 text-center text-xs text-[#717684] gap-3">
              <span className="text-2xl">📭</span>
              <span>No emails found for this filter.</span>
              <button
                onClick={() => {
                  setTimeframe("all");
                  setStatusFilter("all");
                  setActiveCategory("ALL");
                  setSearchQuery("");
                }}
                className="rounded-lg bg-[#1a1e28] px-3 py-1.5 text-xs text-white border border-[#2a2f3e] hover:bg-[#232838] transition"
              >
                Clear all filters
              </button>
            </div>
          ) : (
            emails.map((email) => {
              const isSelected = selectedEmail?.gmail_id === email.gmail_id;
              const cat = categoryConfig[email.category] || categoryConfig.OTHER;
              const pColor = priorityConfig[email.priority] || priorityConfig.LOW;
              const deadlineInfo = formatRelativeDeadline(email.deadline);

              return (
                <div
                  key={email.gmail_id}
                  onClick={() => {
                    setSelectedId(email.gmail_id);
                    setMobileView("reader");
                  }}
                  className={`group relative flex cursor-pointer gap-2.5 rounded-xl p-3 transition border ${
                    isSelected
                      ? "bg-[#181b24] text-white border-amber-500/40 shadow-md"
                      : email.is_read
                      ? "bg-[#101217]/80 text-[#8e93a2] border-transparent hover:bg-[#141720] hover:border-[#1e222d]"
                      : "bg-[#13161f] text-[#d6d9e0] border-[#1e2330] hover:bg-[#171b26]"
                  }`}
                >
                  {/* Left Column: Read Dot / Replied / Star */}
                  <div className="flex flex-col items-center justify-between pt-0.5 shrink-0">
                    <div className="flex flex-col items-center gap-1.5">
                      {!email.is_read && (
                        <span
                          className="h-2 w-2 rounded-full bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.8)]"
                          title="Unread"
                        />
                      )}
                      {email.is_replied && (
                        <span className="text-[11px]" title="Replied">
                          ↩️
                        </span>
                      )}
                    </div>

                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleUpdateStatus(email.gmail_id, { is_starred: !email.is_starred });
                      }}
                      className={`text-xs transition mt-2 ${
                        email.is_starred ? "text-amber-400" : "text-[#3f4350] group-hover:text-[#717684] hover:text-amber-300"
                      }`}
                      title={email.is_starred ? "Unstar" : "Star"}
                    >
                      {email.is_starred ? "★" : "☆"}
                    </button>
                  </div>

                  {/* Avatar with category icon */}
                  <div className="relative shrink-0 pt-0.5">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-[#1e222d] text-[11px] font-bold text-white border border-[#2b303f]">
                      {getInitials(email.sender)}
                    </div>
                    <span
                      className={`absolute -bottom-1 -right-1 flex h-3.5 w-3.5 items-center justify-center rounded-full text-[8px] ${cat.bg} border border-[#101217]`}
                    >
                      {cat.icon}
                    </span>
                  </div>

                  {/* Content summary */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between mb-0.5">
                      <div className="flex items-center gap-1.5 truncate">
                        <span className={`truncate text-[12px] ${!email.is_read ? "font-bold text-white" : "font-medium text-[#c4c7d0]"}`}>
                          {getCleanSenderName(email.sender)}
                        </span>
                      </div>
                      <span className="shrink-0 text-[10px] text-[#636878]">
                        {mounted ? formatShortDate(email.received_at) : ""}
                      </span>
                    </div>

                    <div className={`truncate text-[12px] mb-0.5 ${!email.is_read ? "font-semibold text-white" : "font-normal text-[#a6abb8]"}`}>
                      {email.subject || "(No Subject)"}
                    </div>

                    <div className="truncate text-[11px] text-[#717684] leading-relaxed mb-2">
                      {email.summary || email.snippet}
                    </div>

                    {/* Metadata Pill Row */}
                    <div className="flex flex-wrap items-center justify-between gap-1 text-[10px]">
                      <div className="flex items-center gap-1.5 flex-wrap">
                        {/* Category Badge */}
                        <span className={`rounded-md px-1.5 py-0.5 font-semibold text-[9px] border ${cat.bg} ${cat.text} ${cat.border}`}>
                          {cat.icon} {email.category}
                        </span>

                        {/* Action Badge */}
                        {email.action_required && (
                          <span className="rounded-md bg-amber-500/20 text-amber-300 px-1.5 py-0.5 font-bold border border-amber-500/30">
                            ⚡ Action
                          </span>
                        )}

                        {/* Deadline Indicator */}
                        {deadlineInfo && (
                          <span
                            className={`rounded-md px-1.5 py-0.5 font-semibold text-[9px] border ${
                              deadlineInfo.urgent
                                ? "bg-rose-500/20 text-rose-300 border-rose-500/40 animate-pulse-subtle"
                                : "bg-[#1c202a] text-[#8e93a2] border-[#2b303f]"
                            }`}
                          >
                            ⏰ {deadlineInfo.text}
                          </span>
                        )}
                      </div>

                      {/* Attention Score Badge */}
                      <span
                        className={`font-mono font-bold px-1.5 py-0.5 rounded text-[10px] ${
                          email.attention_score >= 80
                            ? "bg-amber-500/25 text-amber-300 border border-amber-500/40"
                            : email.attention_score >= 50
                            ? "bg-[#1e2330] text-white border border-[#2c3244]"
                            : "text-[#636878]"
                        }`}
                      >
                        {email.attention_score} pts
                      </span>
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </section>

      {/* ========================================================================= */}
      {/* COLUMN 3: EMAIL READER PANE */}
      {/* ========================================================================= */}
      <main
        className={`flex flex-1 flex-col bg-[#0e1014] overflow-hidden ${
          mobileView === "list" ? "hidden md:flex" : "flex w-full"
        }`}
      >
        {selectedEmail ? (
          <>
            {/* Top Reader Action Bar */}
            <header className="flex h-13 items-center justify-between border-b border-[#1a1c23] px-4 md:px-6 text-xs bg-[#101217]/50 shrink-0">
              <div className="flex items-center gap-2">
                {/* Mobile Back to List Button */}
                <button
                  onClick={() => setMobileView("list")}
                  className="flex items-center gap-1 rounded-lg border border-[#242835] bg-[#141720] px-2.5 py-1.5 font-bold text-white md:hidden"
                >
                  <span>← Back</span>
                </button>

                {/* Read / Unread toggle */}
                <button
                  onClick={() => handleUpdateStatus(selectedEmail.gmail_id, { is_read: !selectedEmail.is_read })}
                  className={`flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 font-medium transition ${
                    selectedEmail.is_read
                      ? "border-[#242835] bg-[#141720] text-[#8e93a2] hover:text-white"
                      : "border-blue-500/40 bg-blue-500/15 text-blue-300 font-bold"
                  }`}
                >
                  <span>{selectedEmail.is_read ? "✉️ Mark Unread" : "👁️ Mark Read"}</span>
                </button>

                {/* Replied toggle */}
                <button
                  onClick={() => handleUpdateStatus(selectedEmail.gmail_id, { is_replied: !selectedEmail.is_replied })}
                  className={`hidden sm:flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 font-medium transition ${
                    selectedEmail.is_replied
                      ? "border-emerald-500/40 bg-emerald-500/15 text-emerald-300 font-bold"
                      : "border-[#242835] bg-[#141720] text-[#8e93a2] hover:text-white"
                  }`}
                >
                  <span>↩️ {selectedEmail.is_replied ? "Replied" : "Mark Replied"}</span>
                </button>

                {/* Star toggle */}
                <button
                  onClick={() => handleUpdateStatus(selectedEmail.gmail_id, { is_starred: !selectedEmail.is_starred })}
                  className={`flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 font-medium transition ${
                    selectedEmail.is_starred
                      ? "border-amber-500/40 bg-amber-500/15 text-amber-300 font-bold"
                      : "border-[#242835] bg-[#141720] text-[#8e93a2] hover:text-white"
                  }`}
                >
                  <span>{selectedEmail.is_starred ? "★" : "☆"}</span>
                  <span className="hidden sm:inline">{selectedEmail.is_starred ? "Favorited" : "Favorite"}</span>
                </button>
              </div>

              <div className="flex items-center gap-2">
                <a
                  href={`https://mail.google.com/mail/u/0/#inbox/${selectedEmail.gmail_id}`}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1.5 rounded-lg border border-amber-500/30 bg-gradient-to-r from-amber-500/15 to-orange-500/15 px-3 py-1.5 font-semibold text-amber-300 hover:from-amber-500/25 hover:to-orange-500/25 transition shadow-sm"
                >
                  <span>Open in Gmail</span>
                  <span className="text-[10px]">↗</span>
                </a>
              </div>
            </header>

            {/* Email View Scroll Area */}
            <div className="flex-1 overflow-y-auto px-4 sm:px-6 md:px-8 lg:px-10 py-6 w-full space-y-6">
              {/* Subject Title & Tags */}
              <div className="border-b border-[#1a1c23] pb-5">
                <div className="flex items-start gap-3">
                  <span
                    className={`mt-2 h-3 w-3 shrink-0 rounded-full ${
                      priorityConfig[selectedEmail.priority]?.dot || "bg-zinc-500"
                    }`}
                  />
                  <div className="flex-1">
                    <h1 className="text-lg sm:text-xl font-bold tracking-tight text-white leading-snug">
                      {selectedEmail.subject || "(No Subject)"}
                    </h1>

                    <div className="mt-3 flex flex-wrap items-center gap-2 text-xs">
                      {/* Category Tag */}
                      <span className={`rounded-full px-3 py-0.5 font-semibold text-[11px] border ${categoryConfig[selectedEmail.category]?.bg} ${categoryConfig[selectedEmail.category]?.text} ${categoryConfig[selectedEmail.category]?.border}`}>
                        {categoryConfig[selectedEmail.category]?.icon} {selectedEmail.category}
                      </span>

                      {/* Priority Tag */}
                      <span className={`rounded-full px-2.5 py-0.5 font-bold text-[10px] border ${priorityConfig[selectedEmail.priority]?.badge}`}>
                        {selectedEmail.priority} Priority
                      </span>

                      {/* Attention Score */}
                      <span className="rounded-full bg-[#1a1e28] px-2.5 py-0.5 font-mono text-amber-400 font-bold text-[11px] border border-[#2b303f]">
                        Score: {selectedEmail.attention_score} pts
                      </span>

                      {/* Action Required */}
                      {selectedEmail.action_required && (
                        <span className="rounded-full bg-amber-500/20 text-amber-300 px-2.5 py-0.5 font-bold text-[10px] border border-amber-500/40">
                          ⚡ Action Required
                        </span>
                      )}

                      {/* Status Badges */}
                      {selectedEmail.is_replied && (
                        <span className="rounded-full bg-emerald-500/20 text-emerald-300 px-2.5 py-0.5 font-bold text-[10px] border border-emerald-500/40">
                          ↩️ Replied
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {/* AI Executive Summary Card */}
              {selectedEmail.summary && (
                <div className="rounded-2xl border border-amber-500/30 bg-gradient-to-br from-[#241a12] via-[#1a1410] to-[#120f0d] p-4 sm:p-5 shadow-lg">
                  <div className="mb-2 flex items-center justify-between flex-wrap gap-2">
                    <div className="flex items-center gap-2 text-xs font-bold text-amber-400">
                      <span>✨</span>
                      <span>AI Executive Summary (Ollama Triage)</span>
                    </div>
                    {selectedEmail.deadline && (
                      <div className="inline-flex items-center gap-1.5 rounded-md bg-[#382618] px-2.5 py-1 text-xs font-bold text-amber-300 border border-amber-500/40">
                        <span>⏰ Deadline:</span>
                        <span>{new Date(selectedEmail.deadline).toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}</span>
                      </div>
                    )}
                  </div>
                  <p className="text-[13px] text-[#fef3c7] leading-relaxed font-normal">
                    {selectedEmail.summary}
                  </p>
                </div>
              )}

              {/* Sender & Recipient Information */}
              <div className="flex items-start justify-between rounded-xl border border-[#1a1c23] bg-[#12141a] p-4 flex-wrap gap-2">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#1e222d] text-sm font-bold text-white border border-[#2c3244]">
                    {getInitials(selectedEmail.sender)}
                  </div>
                  <div className="min-w-0">
                    <div className="font-semibold text-white text-[13px] truncate">
                      {getCleanSenderName(selectedEmail.sender)}
                    </div>
                    <div className="text-xs text-[#717684] truncate">
                      {getSenderEmail(selectedEmail.sender)} <span className="text-[#4b4f5c]">→ {selectedEmail.recipient || "me"}</span>
                    </div>
                  </div>
                </div>

                <div className="text-left sm:text-right text-xs text-[#717684]">
                  <div>
                    {new Date(selectedEmail.received_at).toLocaleDateString(undefined, {
                      weekday: "short",
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    })}
                  </div>
                  <div className="text-[11px] text-[#555a66] mt-0.5">
                    {new Date(selectedEmail.received_at).toLocaleTimeString(undefined, {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </div>
                </div>
              </div>

              {/* Email Body Content */}
              <div className="rounded-xl border border-[#1a1c23] bg-[#101217] p-4 sm:p-5">
                <h3 className="text-xs font-bold uppercase tracking-wider text-[#555a66] mb-3">Email Message</h3>
                <div className="text-[13px] leading-relaxed text-[#c6c9cf] whitespace-pre-wrap font-sans break-words">
                  {selectedEmail.body || selectedEmail.snippet}
                </div>
              </div>

              {/* Footer Quick Actions */}
              <div className="flex items-center justify-between rounded-xl border border-[#1a1c23] bg-[#12141a] p-4 flex-wrap gap-3">
                <div className="flex items-center gap-2 text-xs text-[#717684]">
                  <span>💡 Direct reply &amp; smart compose are available directly in Gmail.</span>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleUpdateStatus(selectedEmail.gmail_id, { is_replied: !selectedEmail.is_replied })}
                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-semibold transition ${
                      selectedEmail.is_replied
                        ? "border-emerald-500/40 bg-emerald-500/15 text-emerald-300 font-bold"
                        : "border-[#242835] bg-[#141720] text-[#8e93a2] hover:text-white"
                    }`}
                  >
                    <span>↩️ {selectedEmail.is_replied ? "Marked Replied" : "Mark Replied"}</span>
                  </button>

                  <a
                    href={`https://mail.google.com/mail/u/0/#inbox/${selectedEmail.gmail_id}`}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1.5 rounded-lg bg-gradient-to-r from-amber-500 to-orange-500 px-4 py-1.5 text-xs font-bold text-white shadow hover:from-amber-600 hover:to-orange-600 transition"
                  >
                    <span>Reply in Gmail</span>
                    <span className="text-[10px]">↗</span>
                  </a>
                </div>
              </div>
            </div>
          </>
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center text-[#555a66] gap-2 p-4 text-center">
            <span className="text-4xl">✉️</span>
            <p className="text-xs font-medium">Select an email from the left to view details and AI triage</p>
          </div>
        )}
      </main>
    </div>
  );
}