// stats.ts — pure computation helpers used by StatsPanel.
// Extracted for testability: no React, no DOM, no side effects.

import type { WireUsage } from "./types";

// ─── price table ───────────────────────────────────────────────


// ─── formatting ─────────────────────────────────────────────────

export function fmtTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, "") + "k";
  return String(n);
}

// ─── hit rate ───────────────────────────────────────────────────

export function hitRateColor(rate: number): string {
  return rate >= 80 ? "text-ok" : rate >= 50 ? "text-warning" : "text-err";
}

// ─── Step / Col aggregation ─────────────────────────────────────

export interface StepRecord {
  step: number;
  prompt: number;
  completion: number;
  cacheHit: number;
  cacheMiss: number;
  cost: number;
  source?: string;
}

export interface ColStats {
  prompt: number;
  completion: number;
  cacheHit: number;
  cacheMiss: number;
  cost: number;
}

/** Aggregate a list of steps into column stats (cost summed directly from StepRecord). */
export function aggSteps(steps: StepRecord[]): ColStats {
  let prompt = 0, completion = 0, cacheHit = 0, cacheMiss = 0, cost = 0;
  for (const s of steps) {
    prompt += s.prompt;
    completion += s.completion;
    cacheHit += s.cacheHit;
    cacheMiss += s.cacheMiss;
    cost += s.cost ?? 0;
  }
  return { prompt, completion, cacheHit, cacheMiss, cost };
}

/** Convert a WireUsage snapshot to column stats (cost from costUsd). */
export function colFromUsage(u: WireUsage | undefined): ColStats {
  if (!u) return { prompt: 0, completion: 0, cacheHit: 0, cacheMiss: 0, cost: 0 };
  return { prompt: u.promptTokens, completion: u.completionTokens, cacheHit: u.cacheHitTokens, cacheMiss: u.cacheMissTokens, cost: u.costUsd ?? 0 };
}

// ─── hit rate helpers for StatsPanel ────────────────────────────
