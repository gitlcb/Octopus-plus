'use client';

import { useTranslations } from 'next-intl';
import { Activity, DollarSign, Coins, Timer } from 'lucide-react';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { useStatsData } from './use-stats-data';

export function KpiCards() {
    const t = useTranslations('stats');
    const { total } = useStatsData();

    const requests = formatCount(total.requests).formatted;
    const cost = formatMoney(total.cost).formatted;
    const tokens = formatCount(total.totalToken).formatted;
    const latency = formatTime(total.avgLatency).formatted;

    const cards = [
        {
            icon: Activity,
            label: t('kpi.requests'),
            value: requests.value,
            unit: requests.unit,
            sub: `${t('kpi.successRate')} ${total.successRate.toFixed(1)}%`,
        },
        {
            icon: DollarSign,
            label: t('kpi.cost'),
            value: cost.value,
            unit: cost.unit,
            sub: `${t('kpi.failed')} ${formatCount(total.failed).formatted.value}${formatCount(total.failed).formatted.unit}`,
        },
        {
            icon: Coins,
            label: t('kpi.tokens'),
            value: tokens.value,
            unit: tokens.unit,
            sub: `${t('kpi.cache')} ${formatCount(total.cacheRead + total.cacheWrite).formatted.value}${formatCount(total.cacheRead + total.cacheWrite).formatted.unit}`,
        },
        {
            icon: Timer,
            label: t('kpi.avgLatency'),
            value: latency.value,
            unit: latency.unit,
            sub: `${t('kpi.avgFtut')} ${formatTime(total.avgFtut).formatted.value}${formatTime(total.avgFtut).formatted.unit}`,
        },
    ];

    return (
        <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            {cards.map((c) => (
                <div
                    key={c.label}
                    className="rounded-2xl bg-card border-card-border border text-card-foreground custom-shadow p-4"
                >
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <c.icon className="size-4" />
                        <span className="text-xs">{c.label}</span>
                    </div>
                    <p className="mt-2 text-2xl md:text-3xl font-semibold tabular-nums tracking-tight">
                        <AnimatedNumber value={c.value} />
                        {c.unit && <span className="ml-1 text-base text-muted-foreground">{c.unit}</span>}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground tabular-nums">{c.sub}</p>
                </div>
            ))}
        </section>
    );
}
