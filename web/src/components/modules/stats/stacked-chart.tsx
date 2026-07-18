'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import dayjs from 'dayjs';
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useStatsViewStore } from './store';
import { useStatsData, metricValue, CHART_COLORS } from './use-stats-data';
import { ymdToDayjs } from './trend-chart';

function fmt(v: number, metric: string): string {
    if (metric === 'cost') return `${formatMoney(v).formatted.value}${formatMoney(v).formatted.unit}`;
    if (metric === 'latency') return `${formatTime(v).formatted.value}${formatTime(v).formatted.unit}`;
    return `${formatCount(v).formatted.value}${formatCount(v).formatted.unit}`;
}

export function StackedChart() {
    const t = useTranslations('stats');
    const metric = useStatsViewStore((s) => s.metric);
    const { bucketMap, bucketOrder, dimKeys, dimLabels, bucket } = useStatsData();

    const { chartData, config } = useMemo(() => {
        const cfg: ChartConfig = {};
        dimKeys.forEach((k, i) => {
            cfg[k] = { label: dimLabels.get(k) ?? k, color: CHART_COLORS[i % CHART_COLORS.length] };
        });
        const data = bucketOrder.map((b) => {
            const dm = bucketMap.get(b.k);
            const row: Record<string, string | number> = {
                label: bucket === 'hour' ? dayjs.unix(b.time).format('HH:00') : ymdToDayjs(b.k).format('MM/DD'),
            };
            for (const k of dimKeys) {
                const a = dm?.get(k);
                row[k] = a ? metricValue(a, metric) : 0;
            }
            return row;
        });
        return { chartData: data, config: cfg };
    }, [bucketMap, bucketOrder, dimKeys, dimLabels, bucket, metric]);

    if (dimKeys.length === 0) return null;

    return (
        <section className="rounded-3xl bg-card border-card-border border text-card-foreground custom-shadow p-5">
            <h3 className="text-sm font-medium mb-4">{t('chart.stacked')}</h3>
            <ChartContainer config={config} className="h-64 w-full">
                <BarChart accessibilityLayer data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={24} />
                    <YAxis tickLine={false} axisLine={false} width={48} tickFormatter={(v) => fmt(v, metric)} />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    {dimKeys.map((k, i) => (
                        <Bar
                            key={k}
                            dataKey={k}
                            stackId="a"
                            fill={CHART_COLORS[i % CHART_COLORS.length]}
                            radius={i === dimKeys.length - 1 ? [4, 4, 0, 0] : [0, 0, 0, 0]}
                        />
                    ))}
                </BarChart>
            </ChartContainer>
        </section>
    );
}
