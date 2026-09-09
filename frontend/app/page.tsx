"use client";

import { useEffect, useState } from "react";

type Priority = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

interface RankedEmail {
  gmail_id: string;
  thread_id: string;
  sender: string;
  recipient: string;
  subject: string;
  received_at: string;
  snippet: string;
  category: string;
  priority: Priority;
  summary: string;
  action_required: boolean;
  deadline: string | null;
  attention_score: number;
  ai_processed_at: string | null;
}

const priorityColors: Record<string, { dot: string; text: string }> = {
  CRITICAL: { dot: "bg-rose-500", text: "text-rose-400" },
  HIGH: { dot: "bg-amber-500", text: "text-amber-400" },
  MEDIUM: { dot: "bg-yellow-500", text: "text-yellow-400" },
  LOW: { dot: "bg-emerald-500", text: "text-emerald-400" },
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
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m`;
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h`;
  if (diffInSeconds < 172800) return "Yesterday";
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export default function Home() {
  const [emails, setEmails] = useState<RankedEmail[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [activeNav, setActiveNav] = useState<string>("Priority");
  const [activeHub, setActiveHub] = useState<string>("ALL");
  const [searchQuery, setSearchQuery] = useState("");
  const [starredIds, setStarredIds] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchEmails = async (triggerSync = false) => {
    try {
      setLoading(true);
      setError(null);

      if (triggerSync) {
        try {
          await fetch(`${API_BASE_URL}/api/sync`, { method: "POST" });
        } catch (e) {
          console.warn("Sync trigger failed:", e);
        }
      }

      const res = await fetch(`${API_BASE_URL}/api/emails/priority?limit=50`);
      if (!res.ok) {
        throw new Error(`API error (${res.status})`);
      }
      const data: RankedEmail[] = await res.json();
      setEmails(data || []);
      if (data && data.length > 0 && !selectedId) {
        setSelectedId(data[0].gmail_id);
      }

      // If initially empty and sync was requested, re-check in 4 seconds
      if (triggerSync && (!data || data.length === 0)) {
        setTimeout(async () => {
          try {
            const retryRes = await fetch(`${API_BASE_URL}/api/emails/priority?limit=50`);
            if (retryRes.ok) {
              const retryData: RankedEmail[] = await retryRes.json();
              if (retryData && retryData.length > 0) {
                setEmails(retryData);
                setSelectedId(retryData[0].gmail_id);
              }
            }
          } catch (e) {
            // silent retry
          }
        }, 4000);
      }
    } catch (err: any) {
      console.error("Failed to fetch priority emails:", err);
      setError(`Unable to connect to API server (${API_BASE_URL})`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEmails(true);
  }, []);

  // Filtered emails list
  const filteredEmails = emails.filter((e) => {
    // 1. Sidebar Nav
    if (activeNav === "Starred" && !starredIds[e.gmail_id]) return false;
    if (activeNav === "Action Required" && !e.action_required) return false;

    // 2. Hub Filter
    if (activeHub !== "ALL" && e.category?.toUpperCase() !== activeHub.toUpperCase()) {
      return false;
    }

    // 3. Search Query
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchSubject = e.subject?.toLowerCase().includes(q);
      const matchSender = e.sender?.toLowerCase().includes(q);
      const matchSummary = e.summary?.toLowerCase().includes(q);
      if (!matchSubject && !matchSender && !matchSummary) return false;
    }

    return true;
  });

  const selectedEmail = emails.find((e) => e.gmail_id === selectedId) || filteredEmails[0] || null;

  const leoCount = emails.filter((e) => e.category?.toUpperCase() === "LEO").length;
  const ieeeCount = emails.filter((e) => e.category?.toUpperCase() === "IEEE").length;
  const uniCount = emails.filter((e) => e.category?.toUpperCase() === "UNI").length;
  const actionCount = emails.filter((e) => e.action_required).length;

  const toggleStar = (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    setStarredIds((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#0d0e11] font-sans text-[#e4e5e7] antialiased selection:bg-[#32363f]">
      {/* ========================================================================= */}
      {/* COLUMN 1: LEFT SIDEBAR (Client & Accounts) */}
      {/* ========================================================================= */}
      <aside className="flex w-[240px] shrink-0 flex-col border-r border-[#1e2025] bg-[#121316] p-4 text-xs select-none">
        {/* macOS Window Controls */}
        <div className="mb-5 flex items-center gap-2">
          <div className="h-3 w-3 rounded-full bg-[#ff5f57]" />
          <div className="h-3 w-3 rounded-full bg-[#febc2e]" />
          <div className="h-3 w-3 rounded-full bg-[#28c840]" />
        </div>

        {/* User Profile */}
        <div className="mb-5 flex items-center gap-3 rounded-xl p-1.5 transition hover:bg-[#1a1c21]">
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-tr from-amber-600 to-orange-400 font-semibold text-white shadow-inner">
            MR
          </div>
          <div className="min-w-0 flex-1">
            <div className="truncate font-medium text-white text-[13px]">
              Mesith Rathnayake
            </div>
            <div className="truncate text-[11px] text-[#787c87]">
              {emails.find((e) => e.recipient)?.recipient || "Gmail Account"}
            </div>
          </div>
        </div>

        {/* Search Bar */}
        <div className="relative mb-5">
          <span className="absolute left-2.5 top-2 text-[11px] text-[#787c87]">🔍</span>
          <input
            type="text"
            placeholder="Search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8 w-full rounded-lg bg-[#1a1c21] pl-7 pr-7 text-xs text-white placeholder-[#787c87] focus:outline-none focus:ring-1 focus:ring-[#373a44]"
          />
          <span className="absolute right-2.5 top-2 text-[10px] text-[#555a64] font-mono">⌘K</span>
        </div>

        {/* Main Navigation */}
        <div className="space-y-0.5">
          {[
            { key: "Priority", name: "Priority Inbox", icon: "⭐", count: emails.length },
            { key: "Action Required", name: "Action Items", icon: "⚡", count: actionCount, badgeColor: "bg-amber-500/20 text-amber-300" },
            { key: "Starred", name: "Favorites", icon: "📌", count: Object.values(starredIds).filter(Boolean).length },
            { key: "All", name: "All Mail", icon: "📥", count: emails.length },
          ].map((item) => (
            <button
              key={item.key}
              onClick={() => {
                setActiveNav(item.key);
                setActiveHub("ALL");
              }}
              className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 transition ${
                activeNav === item.key && activeHub === "ALL"
                  ? "bg-[#212329] font-medium text-white shadow-sm"
                  : "text-[#8e929b] hover:bg-[#18191d] hover:text-[#d1d3d8]"
              }`}
            >
              <div className="flex items-center gap-2.5">
                <span className="text-[13px] opacity-80">{item.icon}</span>
                <span>{item.name}</span>
              </div>
              {item.count > 0 && (
                <span
                  className={`rounded-full px-2 py-0.2 text-[10px] font-bold ${
                    item.badgeColor || (activeNav === item.key ? "bg-[#2c2f38] text-white" : "text-[#787c87]")
                  }`}
                >
                  {item.count}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Hubs / Folders Section */}
        <div className="mt-6">
          <div className="mb-2 flex items-center justify-between px-1 text-[10px] font-semibold uppercase tracking-wider text-[#5a5f6b]">
            <span>Hubs & Focus</span>
            <span className="text-xs">▾</span>
          </div>

          <div className="space-y-0.5">
            {[
              { key: "LEO", name: "Leo Club", color: "bg-amber-400", count: leoCount },
              { key: "IEEE", name: "IEEE Branch", color: "bg-sky-400", count: ieeeCount },
              { key: "UNI", name: "Faculty / Uni", color: "bg-purple-400", count: uniCount },
              { key: "JOB", name: "Internships", color: "bg-emerald-400" },
              { key: "SECURITY", name: "Security & 2FA", color: "bg-rose-400" },
            ].map((hub) => (
              <button
                key={hub.key}
                onClick={() => {
                  setActiveHub(hub.key);
                  setActiveNav(hub.name);
                }}
                className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 transition ${
                  activeHub === hub.key
                    ? "bg-[#212329] font-medium text-white shadow-sm"
                    : "text-[#8e929b] hover:bg-[#18191d] hover:text-[#d1d3d8]"
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <span className={`h-2 w-2 rounded-full ${hub.color}`} />
                  <span>{hub.name}</span>
                </div>
                {hub.count !== undefined && hub.count > 0 && (
                  <span className="rounded-full bg-[#1e2026] px-1.5 text-[10px] font-semibold text-[#8e929b]">
                    {hub.count}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>

        {/* Bottom Sync / Server status */}
        <div className="mt-auto border-t border-[#1c1d22] pt-3">
          <button
            onClick={() => fetchEmails(true)}
            disabled={loading}
            className="flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-[#787c87] hover:bg-[#18191d] hover:text-white transition"
          >
            <div className="flex items-center gap-2">
              <span className={`h-2 w-2 rounded-full ${error ? "bg-red-500" : "bg-emerald-400"}`} />
              <span>{loading ? "Syncing..." : "Live Pipeline"}</span>
            </div>
            <span className={loading ? "animate-spin" : ""}>↻</span>
          </button>
        </div>
      </aside>

      {/* ========================================================================= */}
      {/* COLUMN 2: EMAIL LIST (Middle Pane) */}
      {/* ========================================================================= */}
      <section className="flex w-[380px] shrink-0 flex-col border-r border-[#1e2025] bg-[#16171b]">
        {/* Middle Header */}
        <div className="flex items-center justify-between border-b border-[#1e2025] px-4 py-3">
          <div className="flex items-center gap-2">
            <h2 className="text-[14px] font-semibold text-white">
              {activeNav === "Priority" ? "Inbox - Priority" : activeNav}
            </h2>
          </div>
          <button
            onClick={() => fetchEmails(true)}
            disabled={loading}
            className="flex items-center gap-1.5 rounded-lg border border-[#262830] bg-[#1a1c21] px-2.5 py-1 text-[11px] font-medium text-[#8e929b] hover:text-white hover:border-[#3a3d47] transition disabled:opacity-50"
            title="Sync Latest Emails from Gmail"
          >
            <span className={loading ? "animate-spin" : ""}>↻</span>
            <span>{loading ? "Syncing..." : "Sync"}</span>
          </button>
        </div>

        {/* Category Filter Pills */}
        <div className="flex items-center gap-1.5 overflow-x-auto border-b border-[#1e2025] px-4 py-2 text-xs no-scrollbar">
          {[
            { key: "ALL", label: "All" },
            { key: "LEO", label: "🦁 Leo" },
            { key: "IEEE", label: "⚡ IEEE" },
            { key: "UNI", label: "🎓 Uni" },
            { key: "JOB", label: "💼 Jobs" },
            { key: "SECURITY", label: "🔒 Security" },
          ].map((pill) => (
            <button
              key={pill.key}
              onClick={() => setActiveHub(pill.key)}
              className={`rounded-full px-3 py-1 text-[11px] font-medium transition shrink-0 ${
                activeHub === pill.key
                  ? "bg-[#2563eb] text-white font-semibold shadow-sm"
                  : "bg-[#1f2127] text-[#8e929b] hover:text-[#e4e5e7]"
              }`}
            >
              {pill.label}
            </button>
          ))}
        </div>

        {/* Email Cards Stream */}
        <div className="flex-1 overflow-y-auto px-2 space-y-1 py-2">
          {loading && emails.length === 0 ? (
            <div className="p-8 text-center text-xs text-[#787c87]">
              <div className="mb-2 animate-spin text-lg">↻</div>
              Loading &amp; syncing emails...
            </div>
          ) : emails.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-8 text-center text-xs text-[#787c87] gap-3">
              <span>Inbox is empty. Click sync to ingest emails from Gmail.</span>
              <button
                onClick={() => fetchEmails(true)}
                disabled={loading}
                className="rounded-lg bg-[#2563eb] px-3.5 py-1.5 font-medium text-white shadow-sm hover:bg-blue-600 transition disabled:opacity-50"
              >
                {loading ? "Syncing..." : "Sync from Gmail"}
              </button>
            </div>
          ) : filteredEmails.length === 0 ? (
            <div className="p-8 text-center text-xs text-[#787c87]">
              No emails in this view.
            </div>
          ) : (
            filteredEmails.map((email) => {
              const isSelected = selectedEmail?.gmail_id === email.gmail_id;
              const isStarred = !!starredIds[email.gmail_id];
              const pColor = priorityColors[email.priority] || priorityColors.LOW;

              return (
                <div
                  key={email.gmail_id}
                  onClick={() => setSelectedId(email.gmail_id)}
                  className={`group relative flex cursor-pointer gap-3 rounded-xl p-3 transition ${
                    isSelected
                      ? "bg-[#23252c] text-white shadow-sm border border-[#2e313a]"
                      : "hover:bg-[#1c1d22] text-[#c9cbd0]"
                  }`}
                >
                  {/* Star and Status Dot */}
                  <div className="flex flex-col items-center gap-2 pt-0.5">
                    <button
                      onClick={(e) => toggleStar(e, email.gmail_id)}
                      className={`text-xs transition ${
                        isStarred ? "text-amber-400" : "text-[#4b4f5a] hover:text-[#8e929b]"
                      }`}
                    >
                      {isStarred ? "★" : "☆"}
                    </button>
                  </div>

                  {/* Avatar */}
                  <div className="relative shrink-0">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-[#2a2d36] text-[11px] font-semibold text-[#e4e5e7] border border-[#373a44]">
                      {getInitials(email.sender)}
                    </div>
                    {/* Small category dot indicator */}
                    <span
                      className={`absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-[#16171b] ${
                        email.category === "LEO"
                          ? "bg-amber-400"
                          : email.category === "IEEE"
                          ? "bg-sky-400"
                          : email.category === "UNI"
                          ? "bg-purple-400"
                          : pColor.dot
                      }`}
                    />
                  </div>

                  {/* Text Content */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between mb-0.5">
                      <div className="flex items-center gap-1.5 truncate">
                        <span className="truncate font-semibold text-[12px] text-white">
                          {getCleanSenderName(email.sender)}
                        </span>
                        {email.category === "LEO" && (
                          <span className="text-[10px] text-amber-400 font-bold">🦁</span>
                        )}
                      </div>
                      <span className="shrink-0 text-[10px] text-[#6b707c]">
                        {mounted ? formatShortDate(email.received_at) : ""}
                      </span>
                    </div>

                    <div className="truncate text-[12px] font-medium text-[#dce0e8] mb-0.5">
                      {email.subject || "(No Subject)"}
                    </div>

                    <div className="truncate text-[11px] text-[#787c87] leading-relaxed">
                      {email.summary || email.snippet}
                    </div>

                    {/* Meta pill row */}
                    <div className="mt-1.5 flex items-center justify-between text-[10px]">
                      <div className="flex items-center gap-1.5">
                        {email.action_required && (
                          <span className="rounded bg-orange-500/15 px-1.5 py-0.5 font-semibold text-orange-400">
                            Action
                          </span>
                        )}
                        <span className="text-[#555a64] font-medium uppercase text-[9px]">
                          {email.category}
                        </span>
                      </div>

                      {/* Score Badge */}
                      <span
                        className={`font-mono font-bold px-1.5 py-0.2 rounded text-[10px] ${
                          email.attention_score >= 70
                            ? "bg-amber-500/20 text-amber-300"
                            : email.attention_score >= 40
                            ? "bg-white/10 text-white"
                            : "text-[#6b707c]"
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
      {/* COLUMN 3: EMAIL READING & AI SUMMARY PANE (Right Column) */}
      {/* ========================================================================= */}
      <main className="flex flex-1 flex-col bg-[#121316] overflow-hidden">
        {selectedEmail ? (
          <>
            {/* Action Bar Header */}
            <header className="flex h-12 items-center justify-between border-b border-[#1e2025] px-6 text-[#787c87]">
              <div className="flex items-center gap-3">
                <button
                  onClick={(e) => toggleStar(e, selectedEmail.gmail_id)}
                  className={`flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium transition ${
                    starredIds[selectedEmail.gmail_id]
                      ? "bg-amber-500/10 text-amber-400"
                      : "text-[#8e929b] hover:bg-[#1a1c21] hover:text-white"
                  }`}
                  title="Toggle Favorite"
                >
                  <span>{starredIds[selectedEmail.gmail_id] ? "★" : "☆"}</span>
                  <span>{starredIds[selectedEmail.gmail_id] ? "Favorited" : "Favorite"}</span>
                </button>
              </div>

              <div className="flex items-center gap-3">
                <a
                  href={`https://mail.google.com/mail/u/0/#inbox/${selectedEmail.gmail_id}`}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1.5 rounded-lg border border-[#2a2d36] bg-[#1a1c21] px-3 py-1.5 text-xs font-medium text-[#dce0e8] hover:border-[#3d424f] hover:text-white transition shadow-sm"
                >
                  <span>Open in Gmail</span>
                  <span className="text-[10px] text-[#787c87]">↗</span>
                </a>
              </div>
            </header>

            {/* Email Body Scroll Container */}
            <div className="flex-1 overflow-y-auto px-8 py-6 max-w-4xl">
              {/* Subject Title with Priority Dot */}
              <div className="mb-6 flex items-start gap-3">
                <span
                  className={`mt-2 h-2.5 w-2.5 shrink-0 rounded-full ${
                    priorityColors[selectedEmail.priority]?.dot || "bg-zinc-500"
                  }`}
                />
                <div className="flex-1">
                  <h1 className="text-xl font-bold tracking-tight text-white leading-snug">
                    {selectedEmail.subject || "(No Subject)"}
                  </h1>

                  <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
                    <span className="rounded-full bg-[#1e2129] px-2.5 py-0.5 text-[11px] font-semibold text-[#a8acb6] border border-[#2c303a]">
                      {selectedEmail.category}
                    </span>
                    <span className="text-[#626775]">|</span>
                    <span className="font-mono text-amber-400 font-semibold text-xs">
                      Attention Score: {selectedEmail.attention_score}
                    </span>
                    {selectedEmail.action_required && (
                      <span className="rounded bg-orange-500/15 px-2 py-0.5 text-[10px] font-bold text-orange-400 border border-orange-500/30">
                        ⚡ Action Required
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* AI Executive Summary Card (Matches Image Warm Amber/Brown Container) */}
              {selectedEmail.summary && (
                <div className="mb-6 rounded-2xl border border-[#3b2a1d] bg-[#241a13] p-4 shadow-sm">
                  <div className="mb-2 flex items-center gap-2 text-xs font-bold text-[#d97706]">
                    <span>✨</span>
                    <span>Summary of this email</span>
                  </div>
                  <p className="text-[13px] text-[#fef3c7] leading-relaxed font-normal">
                    {selectedEmail.summary}
                  </p>
                  {selectedEmail.deadline && (
                    <div className="mt-3 inline-flex items-center gap-1.5 rounded-md bg-[#372417] px-2.5 py-1 text-xs font-semibold text-amber-300">
                      <span>⏰ Deadline:</span>
                      <span>{selectedEmail.deadline}</span>
                    </div>
                  )}
                </div>
              )}

              {/* Sender Details Header */}
              <div className="mb-6 flex items-start justify-between border-b border-[#1e2025] pb-5">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[#272a33] text-sm font-semibold text-white border border-[#373b47]">
                    {getInitials(selectedEmail.sender)}
                  </div>
                  <div>
                    <div className="font-semibold text-white text-[13px]">
                      {getCleanSenderName(selectedEmail.sender)}
                    </div>
                    <div className="text-xs text-[#787c87]">
                      {getSenderEmail(selectedEmail.sender)} <span className="text-[#4e535e]">→ to me</span>
                    </div>
                  </div>
                </div>

                <div className="text-right text-xs text-[#6b707c]">
                  {new Date(selectedEmail.received_at).toLocaleString(undefined, {
                    month: "short",
                    day: "numeric",
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </div>
              </div>

              {/* Email Content Snippet & Body */}
              <div className="text-[13px] leading-relaxed text-[#c6c9cf] whitespace-pre-wrap font-sans">
                {selectedEmail.snippet}
              </div>
            </div>
          </>
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center text-[#555a64]">
            <span className="text-3xl mb-2">✉️</span>
            <p className="text-xs">Select an email to view details</p>
          </div>
        )}
      </main>
    </div>
  );
}