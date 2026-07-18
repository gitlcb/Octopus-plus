'use client';

import { useMemo } from 'react';
import { useStatsDimension, type StatsDimRow } from '@/api/endpoints/stats';
import { useStatsViewStore, rangeToWindow, type StatsMetric } from './store';

export const CHART_COLORS = [
    'var(--chart-1)',
    'var(--chart-2)',
    'var(--chart-3)',
    'var(--chart-4)',
    'var(--chart-5)',
];

export interface DimAggregate {
    key: string;
    label: string;
    inputToken: number;
    outputToken: number;
    totalToken: number;
    cost: number;
    waitTime: number;
    success: number;
    failed: number;
    requests: number;
    successRate: number; // 0-100
    avgLatency: number;  // ms/请求
    avgFtut: number;     // ms/成功
    cacheRead: number;
    cacheWrite: number;
}

function emptyAgg(key: string, label: string): DimAggregate {
    return {
        key, label,
        inputToken: 0, outputToken: 0, totalToken: 0, cost: 0, waitTime: 0,
        success: 0, failed: 0, requests: 0, successRate: 0, avgLatency: 0, avgFtut: 0,
        cacheRead: 0, cacheWrite: 0,
    };
}

function accumulate(agg: DimAggregate, r: StatsDimRow) {
    agg.inputToken += r.input_token;
    agg.outputToken += r.output_token;
    agg.totalToken += r.input_token + r.output_token;
    agg.cost += r.input_cost + r.output_cost;
    agg.waitTime += r.wait_time;
    agg.success += r.request_success;
    agg.failed += r.request_failed;
    agg.cacheRead += r.cache_read_token;
    agg.cacheWrite += r.cache_write_token;
}

function finalize(agg: DimAggregate, ftutSum: number) {
    agg.requests = agg.success + agg.failed;
    agg.successRate = agg.requests > 0 ? (agg.success / agg.requests) * 100 : 0;
    agg.avgLatency = agg.requests > 0 ? agg.waitTime / agg.requests : 0;
    agg.avgFtut = agg.success > 0 ? ftutSum / agg.success : 0;
}

/** 指标取值器:把聚合行映射为当前选中指标的数值(趋势/堆叠图 Y 轴)。 */
export function metricValue(agg: DimAggregate, metric: StatsMetric): number {
    switch (metric) {
        case 'cost': return agg.cost;
        case 'requests': return agg.requests;
        case 'tokens': return agg.totalToken;
        case 'latency': return agg.avgLatency;
    }
}

/**
 * 统计页共享数据:一次取分维度按时间桶的数据,派生
 * - total: 全局 KPI 聚合
 * - byDim: 各维度总量(分布/表格,已按费用降序)
 * - series: 时间桶 × 维度(趋势/堆叠图),形如 [{ bucketLabel, [dimKey]: value }]
 */
export function useStatsData() {
    const { range, groupBy } = useStatsViewStore();
    const win = rangeToWindow(range);

    const { data, isLoading } = useStatsDimension({
        groupBy,
        bucket: win.bucket,
        from: win.from,
        to: win.to,
        limit: 8,
    });

    return useMemo(() => {
        const rows = data?.rows ?? [];

        // 全局总量
        const total = emptyAgg('__total__', 'total');
        let totalFtut = 0;
        for (const r of rows) {
            accumulate(total, r);
            totalFtut += r.ftut_time;
        }
        finalize(total, totalFtut);

        // 各维度总量
        const dimMap = new Map<string, DimAggregate>();
        const dimFtut = new Map<string, number>();
        for (const r of rows) {
            let a = dimMap.get(r.key);
            if (!a) { a = emptyAgg(r.key, r.label); dimMap.set(r.key, a); }
            accumulate(a, r);
            dimFtut.set(r.key, (dimFtut.get(r.key) ?? 0) + r.ftut_time);
        }
        const byDim = Array.from(dimMap.values());
        for (const a of byDim) finalize(a, dimFtut.get(a.key) ?? 0);
        byDim.sort((x, y) => y.cost - x.cost);

        // 时间桶 × 维度(用于趋势/堆叠)
        const bucketMap = new Map<string, Map<string, DimAggregate>>();
        const bucketFtut = new Map<string, Map<string, number>>();
        const bucketOrder: { k: string; time: number; label: string }[] = [];
        for (const r of rows) {
            const bk = win.bucket === 'hour' ? String(r.time ?? 0) : (r.date ?? '');
            if (!bucketMap.has(bk)) {
                bucketMap.set(bk, new Map());
                bucketFtut.set(bk, new Map());
                bucketOrder.push({ k: bk, time: r.time ?? 0, label: bk });
            }
            const dm = bucketMap.get(bk)!;
            let a = dm.get(r.key);
            if (!a) { a = emptyAgg(r.key, r.label); dm.set(r.key, a); }
            accumulate(a, r);
            const fm = bucketFtut.get(bk)!;
            fm.set(r.key, (fm.get(r.key) ?? 0) + r.ftut_time);
        }
        for (const [bk, dm] of bucketMap) {
            for (const [dk, a] of dm) finalize(a, bucketFtut.get(bk)!.get(dk) ?? 0);
        }
        bucketOrder.sort((a, b) => a.time - b.time);

        const dimKeys = byDim.map((d) => d.key);
        const dimLabels = new Map(byDim.map((d) => [d.key, d.label]));

        return { total, byDim, bucketMap, bucketOrder, dimKeys, dimLabels, isLoading, bucket: win.bucket };
    }, [data, win.bucket]);
}
