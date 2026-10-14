import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError, exportKeyEvents, fetchKeyEventFilterOptions, fetchKeyEventRequestLog, fetchKeyEvents, isUsageRangeBoundsConflict } from '@/lib/api';
import type { AuthSessionAPIKeySummary, UsageCustomRange, UsageEvent, UsageEventRequestLogResponse, UsageSourceFilterOption, UsageTimeRange } from '@/lib/types';
import { KeyViewerShell } from '@/features/key-viewer/KeyViewerShell';
import type { KeyViewerPath } from '@/features/key-viewer/navigation';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useThemeStore } from '@/stores';
import { TimeRangeControl } from '@/components/usage';
import {
  RequestEventsDetailsCard,
  normalizeRequestEventColumnOrder,
  normalizeRequestEventVisibleColumnIds,
  type RequestEventColumnId,
  type RequestEventExportFormat,
} from '@/components/usage/RequestEventsDetailsCard';
import {
  clampStoredUsageRangeStateToCurrentBounds,
  resolveUsageRangeRecoveryTimeZone,
  type StoredUsageRangeState,
} from '@/utils/usage/customRange';
import { buildUsageRangeQuery } from '@/utils/usage/rangeQuery';
import { loadKeyViewerTimeRange, persistKeyViewerTimeRange } from '@/features/key-viewer/timeRange';
import styles from '@/features/key-viewer/KeyViewerShell.module.scss';

const REQUEST_EVENTS_COLUMN_STORAGE_KEY = 'cli-proxy-usage-key-events-columns-v1';
const REQUEST_EVENTS_COLUMN_ORDER_STORAGE_KEY = 'cli-proxy-usage-key-events-column-order-v1';

const loadTimeRange = (): StoredUsageRangeState => {
  return loadKeyViewerTimeRange();
};

const loadColumnSettings = (): {
  visibleColumnIds: RequestEventColumnId[] | undefined;
  columnOrder: RequestEventColumnId[] | undefined;
} => {
  try {
    const storedVisible = localStorage.getItem(REQUEST_EVENTS_COLUMN_STORAGE_KEY);
    const storedOrder = localStorage.getItem(REQUEST_EVENTS_COLUMN_ORDER_STORAGE_KEY);
    return {
      visibleColumnIds: storedVisible ? normalizeRequestEventVisibleColumnIds(JSON.parse(storedVisible)) : undefined,
      columnOrder: storedOrder ? normalizeRequestEventColumnOrder(JSON.parse(storedOrder)) : undefined,
    };
  } catch {
    return {
      visibleColumnIds: undefined,
      columnOrder: undefined,
    };
  }
};

export interface KeyEventsPageProps {
  apiKey?: AuthSessionAPIKeySummary;
  onNavigate: (path: KeyViewerPath) => void;
  onAuthRequired?: () => void;
}

export function KeyEventsPage({ apiKey, onNavigate, onAuthRequired }: KeyEventsPageProps) {
  const { t } = useTranslation();
  const isMobile = useMediaQuery('(max-width: 768px)');
  const resolvedTheme = useThemeStore((state) => state.resolvedTheme);
  const isDark = resolvedTheme === 'dark';
  const [timeRangeState, setTimeRangeState] = useState<StoredUsageRangeState>(loadTimeRange);
  const { range: timeRange, customRange } = timeRangeState;
  const [events, setEvents] = useState<UsageEvent[]>([]);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [eventsError, setEventsError] = useState('');
  const [modelFilterOptions, setModelFilterOptions] = useState<string[]>([]);
  const [sourceFilterOptions, setSourceFilterOptions] = useState<UsageSourceFilterOption[]>([]);
  const [selectedModel, setSelectedModel] = useState<string>('');
  const [selectedSource, setSelectedSource] = useState<string>('');
  const [selectedResult, setSelectedResult] = useState<string>('');
  const [manualRefreshLoading, setManualRefreshLoading] = useState(false);
  const [columnSettings, setColumnSettings] = useState(loadColumnSettings);
  const [requestLog, setRequestLog] = useState<UsageEventRequestLogResponse | null>(null);
  const [requestLogEventId, setRequestLogEventId] = useState<string | null>(null);
  const [requestLogLoading, setRequestLogLoading] = useState(false);
  const requestControllerRef = useRef<AbortController | null>(null);
  const filterOptionsControllerRef = useRef<AbortController | null>(null);
  const requestLogControllerRef = useRef<AbortController | null>(null);
  const eventsTimeZoneRef = useRef(timeRangeState.timeZone);
  const usageRangeQuery = useMemo(() => buildUsageRangeQuery({
    range: timeRange,
    customUnit: customRange?.unit,
    customStart: customRange?.start,
    customEnd: customRange?.end,
  }), [customRange?.end, customRange?.start, customRange?.unit, timeRange]);
  const rangeTimeZone = timeRangeState.timeZone;

  const recoverRangeBoundsConflict = useCallback((error: unknown) => {
    if (!isUsageRangeBoundsConflict(error)) return false;
    const timeZone = resolveUsageRangeRecoveryTimeZone(timeRangeState, eventsTimeZoneRef.current)?.trim();
    if (!timeZone) return false;
    const nextState = clampStoredUsageRangeStateToCurrentBounds(timeRangeState, {
      nowMs: Date.now(),
      timeZone,
    });
    if (nextState === timeRangeState) return false;
    setTimeRangeState(nextState);
    return true;
  }, [timeRangeState]);

  const handleTimeRangeChange = useCallback((range: UsageTimeRange, nextCustomRange?: UsageCustomRange) => {
    let nextState: StoredUsageRangeState;
    if (range === 'custom' && nextCustomRange) {
      nextState = { range, customRange: nextCustomRange, timeZone: rangeTimeZone };
    } else {
      nextState = { ...timeRangeState, range };
    }
    setTimeRangeState(nextState);
    persistKeyViewerTimeRange(nextState);
  }, [rangeTimeZone, timeRangeState]);

  const loadFilterOptions = useCallback(async () => {
    filterOptionsControllerRef.current?.abort();
    const controller = new AbortController();
    filterOptionsControllerRef.current = controller;
    try {
      const options = await fetchKeyEventFilterOptions(controller.signal);
      if (filterOptionsControllerRef.current !== controller) return;
      setModelFilterOptions(options.models || []);
      setSourceFilterOptions(options.sources || []);
    } catch (error) {
      if (controller.signal.aborted || filterOptionsControllerRef.current !== controller) return;
      if (error instanceof ApiError && error.status === 401) {
        onAuthRequired?.();
        return;
      }
      // Silently fail for filter options
    } finally {
      if (filterOptionsControllerRef.current === controller) {
        filterOptionsControllerRef.current = null;
      }
    }
  }, [onAuthRequired]);

  const loadEvents = useCallback(async () => {
    if (!usageRangeQuery.valid) return;
    requestControllerRef.current?.abort();
    const controller = new AbortController();
    requestControllerRef.current = controller;
    setEventsLoading(true);
    setEventsError('');

    try {
      const response = await fetchKeyEvents(usageRangeQuery, controller.signal, {
        model: selectedModel || undefined,
        source: selectedSource || undefined,
        result: selectedResult || undefined,
      });
      if (requestControllerRef.current !== controller) return;
      eventsTimeZoneRef.current = timeRangeState.timeZone;
      setEvents(response.events || []);
      setEventsTotal(response.total_count || 0);
      setEventsLoading(false);
    } catch (error) {
      if (controller.signal.aborted || requestControllerRef.current !== controller) return;
      setEventsLoading(false);
      if (recoverRangeBoundsConflict(error)) return;
      if (error instanceof ApiError && error.status === 401) {
        onAuthRequired?.();
        return;
      }
      setEventsError('KEY_EVENTS_LOAD_FAILED');
    } finally {
      if (requestControllerRef.current === controller) {
        requestControllerRef.current = null;
      }
    }
  }, [onAuthRequired, recoverRangeBoundsConflict, selectedModel, selectedResult, selectedSource, timeRangeState.timeZone, usageRangeQuery]);


  useEffect(() => {
    void loadEvents();
    return () => {
      requestControllerRef.current?.abort();
      requestControllerRef.current = null;
    };
  }, [loadEvents]);

  useEffect(() => {
    void loadFilterOptions();
    return () => {
      filterOptionsControllerRef.current?.abort();
      filterOptionsControllerRef.current = null;
    };
  }, [loadFilterOptions]);

  useEffect(() => {
    persistKeyViewerTimeRange(timeRangeState);
  }, [timeRangeState]);

  const handleManualRefresh = useCallback(async () => {
    if (manualRefreshLoading) return;
    setManualRefreshLoading(true);
    try {
      await Promise.all([loadEvents(), loadFilterOptions()]);
    } finally {
      setManualRefreshLoading(false);
    }
  }, [loadEvents, loadFilterOptions, manualRefreshLoading]);

  const handleExport = useCallback(async (format: RequestEventExportFormat) => {
    if (!usageRangeQuery.valid) return;
    try {
      const file = await exportKeyEvents(usageRangeQuery, format, {
        model: selectedModel || undefined,
        source: selectedSource || undefined,
        result: selectedResult || undefined,
      });
      const url = URL.createObjectURL(file.blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = file.filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        onAuthRequired?.();
      }
    }
  }, [onAuthRequired, selectedModel, selectedResult, selectedSource, usageRangeQuery]);

  const handleRequestLogOpen = useCallback(async (event: UsageEvent) => {
    const eventId = event.id;
    if (!eventId) return;
    setRequestLogEventId(eventId);
    setRequestLogLoading(true);
    setRequestLog(null);
    requestLogControllerRef.current?.abort();
    const controller = new AbortController();
    requestLogControllerRef.current = controller;

    try {
      const log = await fetchKeyEventRequestLog(eventId, controller.signal);
      if (requestLogControllerRef.current !== controller) return;
      setRequestLog(log);
      setRequestLogLoading(false);
    } catch (error) {
      if (controller.signal.aborted || requestLogControllerRef.current !== controller) return;
      setRequestLogLoading(false);
      if (error instanceof ApiError && error.status === 401) {
        onAuthRequired?.();
      }
    } finally {
      if (requestLogControllerRef.current === controller) {
        requestLogControllerRef.current = null;
      }
    }
  }, [onAuthRequired]);

  const handleRequestLogClose = useCallback(() => {
    setRequestLogEventId(null);
    setRequestLog(null);
    requestLogControllerRef.current?.abort();
    requestLogControllerRef.current = null;
  }, []);

  const handleVisibleColumnIdsChange = useCallback((columnIds: RequestEventColumnId[]) => {
    setColumnSettings((current) => {
      const next = { ...current, visibleColumnIds: columnIds };
      try {
        localStorage.setItem(REQUEST_EVENTS_COLUMN_STORAGE_KEY, JSON.stringify(columnIds));
      } catch {
        // Ignore storage errors
      }
      return next;
    });
  }, []);

  const handleColumnOrderChange = useCallback((columnIds: RequestEventColumnId[]) => {
    setColumnSettings((current) => {
      const next = { ...current, columnOrder: columnIds };
      try {
        localStorage.setItem(REQUEST_EVENTS_COLUMN_ORDER_STORAGE_KEY, JSON.stringify(columnIds));
      } catch {
        // Ignore storage errors
      }
      return next;
    });
  }, []);

  const displayEventsError = eventsError ? t('key_viewer.events_load_failed') : '';

  return (
    <KeyViewerShell
      activePage="events"
      apiKey={apiKey}
      loading={eventsLoading}
      filters={[
        <TimeRangeControl
          key="range"
          value={timeRange}
          customRange={customRange}
          timeZone={rangeTimeZone}
          ariaLabel={t('usage_stats.range_filter')}
          onChange={handleTimeRangeChange}
        />,
      ]}
      onRefresh={handleManualRefresh}
      refreshing={manualRefreshLoading}
      refreshDisabled={eventsLoading}
      onNavigate={onNavigate}
      onAuthRequired={onAuthRequired}
    >
      <div className={styles.content}>
        <RequestEventsDetailsCard
          events={events}
          loading={eventsLoading}
          totalCount={eventsTotal}
          modelOptions={modelFilterOptions}
          sourceOptions={sourceFilterOptions}
          modelFilter={selectedModel}
          sourceFilter={selectedSource}
          resultFilter={selectedResult}
          onModelFilterChange={setSelectedModel}
          onSourceFilterChange={setSelectedSource}
          onResultFilterChange={setSelectedResult}
          onExport={handleExport}
          requestLogAccessEnabled={true}
          onRequestLogOpen={handleRequestLogOpen}
          requestLogLoadingEventId={requestLogEventId}
          requestLogResponse={requestLog}
          onRequestLogClose={handleRequestLogClose}
          visibleColumnIds={columnSettings.visibleColumnIds}
          columnOrder={columnSettings.columnOrder}
          onVisibleColumnIdsChange={handleVisibleColumnIdsChange}
          onColumnOrderChange={handleColumnOrderChange}
        />
      </div>
    </KeyViewerShell>
  );
}
