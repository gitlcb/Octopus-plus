'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Cell, Pie, PieChart } from 'recharts';
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart';
import {
    Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useStatsData, CHART_COLORS } from './use-stats-data';

export function Breakdown() {
    const t = useTranslations('stats');
    const { byDim } = useStatsData();

    const { pieData, config } = useMemo(() => {
        const cfg: ChartConfig = {};
        const data = byDim.map((d, i) => {
            cfg[d.key] = { label: d.label, color: CHART_COLORS[i % CHART_COLORS.length] };
            return { key: d.key, label: d.label, value: d.cost, fill: CHART_COLORS[i % CHART_COLORS.length] };
        }).filter((d) => d.value > 0);
        return { pieData: data, config: cfg };
    }, [byDim]);

    return (
        <section className="rounded-3xl bg-card border-card-border border text-card-foreground custom-shadow p-5">
            <h3 className="text-sm font-medium mb-4">{t('chart.breakdown')}</h3>
            <div className="grid gap-6 lg:grid-cols-[220px_1fr] items-center">
                {/* 饼图:按费用占比 */}
                <div className="flex justify-center">
                    {pieData.length > 0 ? (
                        <ChartContainer config={config} className="h-52 w-52">
                            <PieChart>
                                <ChartTooltip content={<ChartTooltipContent nameKey="label" />} />
                                <Pie data={pieData} dataKey="value" nameKey="label" innerRadius={44} outerRadius={80} strokeWidth={2}>
                                    {pieData.map((d) => (
                                        <Cell key={d.key} fill={d.fill} />
                                    ))}
                                </Pie>
                            </PieChart>
                        </ChartContainer>
                    ) : (
                        <div className="h-52 grid place-items-center text-sm text-muted-foreground">{t('empty')}</div>
                    )}
                </div>

                {/* 明细表 */}
                <div className="overflow-x-auto">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>{t('table.name')}</TableHead>
                                <TableHead className="text-right">{t('table.requests')}</TableHead>
                                <TableHead className="text-right">{t('table.successRate')}</TableHead>
                                <TableHead className="text-right">{t('table.tokens')}</TableHead>
                                <TableHead className="text-right">{t('table.cost')}</TableHead>
                                <TableHead className="text-right">{t('table.avgLatency')}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {byDim.map((d, i) => {
                                const req = formatCount(d.requests).formatted;
                                const tok = formatCount(d.totalToken).formatted;
                                const cost = formatMoney(d.cost).formatted;
                                const lat = formatTime(d.avgLatency).formatted;
                                return (
                                    <TableRow key={d.key}>
                                        <TableCell className="font-medium">
                                            <span className="inline-flex items-center gap-2">
                                                <span className="size-2.5 rounded-full" style={{ background: CHART_COLORS[i % CHART_COLORS.length] }} />
                                                <span className="truncate max-w-[180px]">{d.label}</span>
                                            </span>
                                        </TableCell>
                                        <TableCell className="text-right tabular-nums">{req.value}{req.unit}</TableCell>
                                        <TableCell className="text-right tabular-nums">{d.successRate.toFixed(1)}%</TableCell>
                                        <TableCell className="text-right tabular-nums">{tok.value}{tok.unit}</TableCell>
                                        <TableCell className="text-right tabular-nums">{cost.value}{cost.unit}</TableCell>
                                        <TableCell className="text-right tabular-nums">{lat.value}{lat.unit}</TableCell>
                                    </TableRow>
                                );
                            })}
                            {byDim.length === 0 && (
                                <TableRow>
                                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">{t('empty')}</TableCell>
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </div>
            </div>
        </section>
    );
}
