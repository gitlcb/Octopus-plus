'use client';

import { useTranslations } from 'next-intl';
import { Tabs, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import {
    Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { useStatsViewStore, type StatsRange, type StatsMetric } from './store';
import type { StatsDimGroupBy } from '@/api/endpoints/stats';

export function FilterBar() {
    const t = useTranslations('stats');
    const { range, groupBy, metric, setRange, setGroupBy, setMetric } = useStatsViewStore();

    return (
        <section className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <Tabs value={range} onValueChange={(v) => setRange(v as StatsRange)}>
                <TabsList>
                    <TabsTrigger value="1">{t('range.today')}</TabsTrigger>
                    <TabsTrigger value="7">{t('range.last7Days')}</TabsTrigger>
                    <TabsTrigger value="30">{t('range.last30Days')}</TabsTrigger>
                    <TabsTrigger value="90">{t('range.last90Days')}</TabsTrigger>
                </TabsList>
            </Tabs>

            <div className="flex items-center gap-2">
                <Select value={groupBy} onValueChange={(v) => setGroupBy(v as StatsDimGroupBy)}>
                    <SelectTrigger className="w-32" size="sm">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="model">{t('dimension.model')}</SelectItem>
                        <SelectItem value="channel">{t('dimension.channel')}</SelectItem>
                        <SelectItem value="apikey">{t('dimension.apikey')}</SelectItem>
                    </SelectContent>
                </Select>

                <Select value={metric} onValueChange={(v) => setMetric(v as StatsMetric)}>
                    <SelectTrigger className="w-28" size="sm">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="cost">{t('metric.cost')}</SelectItem>
                        <SelectItem value="requests">{t('metric.requests')}</SelectItem>
                        <SelectItem value="tokens">{t('metric.tokens')}</SelectItem>
                        <SelectItem value="latency">{t('metric.latency')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>
        </section>
    );
}
