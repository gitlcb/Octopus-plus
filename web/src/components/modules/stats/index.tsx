'use client';

import { PageWrapper } from '@/components/common/PageWrapper';
import { FilterBar } from './filter-bar';
import { KpiCards } from './kpi-cards';
import { TrendChart } from './trend-chart';
import { StackedChart } from './stacked-chart';
import { Breakdown } from './breakdown';

export function Stats() {
    return (
        <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
            <FilterBar />
            <KpiCards />
            <TrendChart />
            <StackedChart />
            <Breakdown />
        </PageWrapper>
    );
}
