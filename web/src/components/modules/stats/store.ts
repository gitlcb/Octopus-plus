'use client';

import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';
import type { StatsDimGroupBy } from '@/api/endpoints/stats';

export type StatsRange = '1' | '7' | '30' | '90';
export type StatsMetric = 'cost' | 'requests' | 'tokens' | 'latency';

interface StatsViewState {
    range: StatsRange;
    groupBy: StatsDimGroupBy;
    metric: StatsMetric;
    setRange: (v: StatsRange) => void;
    setGroupBy: (v: StatsDimGroupBy) => void;
    setMetric: (v: StatsMetric) => void;
}

export const useStatsViewStore = create<StatsViewState>()(
    persist(
        (set) => ({
            range: '7',
            groupBy: 'model',
            metric: 'cost',
            setRange: (v) => set({ range: v }),
            setGroupBy: (v) => set({ groupBy: v }),
            setMetric: (v) => set({ metric: v }),
        }),
        {
            name: 'stats-view-options-storage',
            storage: createJSONStorage(() => localStorage),
        }
    )
);

/** 把 range 档位换算成 from/to(unix 秒)与 bucket 粒度。 */
export function rangeToWindow(range: StatsRange): { from: number; to: number; bucket: 'hour' | 'day' } {
    const now = Math.floor(Date.now() / 1000);
    const days = Number(range);
    return {
        from: now - days * 24 * 3600,
        to: now,
        bucket: range === '1' ? 'hour' : 'day',
    };
}
