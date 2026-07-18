'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import dayjs from 'dayjs';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useStatsViewStore } from './store';
import { useStatsData, metricValue, type DimAggregate } from './use-stats-data';

function formatMetric(v: number, metric: string): string {
    if (metric === 'cost') return `${formatMoney(v).formatted.value}${formatMoney(v).formatted.unit}`;
    if (metric === 'latency') return `${formatTime(v).formatted.value}${formatTime(v).formatted.unit}`;
    return `${formatCount(v).formatted.value}${formatCount(v).formatted.unit}`;
}

/** "20060102" -> dayjs(ISO),避免依赖 customParseFormat 插件。 */
export function ymdToDayjs(s: string) {
    if (s.length === 8) return dayjs(`${s.slice(0, 4)}-${s.slice(4, 6)}-${s.slice(6, 8)}`);
    return dayjs(s);
}

export function TrendChart() {
    const t = useTranslations('stats');
    const metric = useStatsViewStore((s) => s.metric);
    const { bucketMap, bucketOrder, bucket } = useStatsData();

    const chartData = useMemo(() => {
        return bucketOrder.map((b) => {
            const dm = bucketMap.get(b.k);
            let value = 0;
            let waitSum = 0;
            let reqSum = 0;
            if (dm) {
                for (const a of dm.values()) {
                    if (metric === 'latency') {
                        waitSum += a.waitTime;
                        reqSum += a.requests;
                    } else {
                        value += metricValue(a, metric);
                    }
                }
            }
            if (metric === 'latency') value = reqSum > 0 ? waitSum / reqSum : 0;
            const label = bucket === 'hour'
                ? dayjs.unix(b.time).format('HH:00')
                : ymdToDayjs(b.k).format('MM/DD');
            return { label, value };
        });
    }, [bucketMap, bucketOrder, bucket, metric]);

    const chartConfig = useMemo(() => ({ value: { label: t(`metric.${metric}`) } }), [t, metric]);

    return (
        <section className="rounded-3xl bg-card border-card-border border text-card-foreground custom-shadow p-5">
            <h3 className="text-sm font-medium mb-4">{t('chart.trend')}</h3>
            <ChartContainer config={chartConfig} className="h-56 w-full">
                <AreaChart accessibilityLayer data={chartData}>
                    <defs>
                        <linearGradient id="fillTrend" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={0.35} />
                            <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0.05} />
                        </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />
                    <YAxis tickLine={false} axisLine={false} width={48} tickFormatter={(v) => formatMetric(v, metric)} />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                    <Area type="monotone" dataKey="value" stroke="var(--chart-1)" fill="url(#fillTrend)" />
                </AreaChart>
            </ChartContainer>
        </section>
    );
}

// re-export 供其它组件复用类型
export type { DimAggregate };
