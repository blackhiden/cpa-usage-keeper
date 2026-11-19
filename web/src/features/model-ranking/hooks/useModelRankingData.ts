import { useCallback, useEffect, useRef, useState } from 'react';
import { fetchModelRankingLeaderboard, ModelRankingApiError } from '../api';
import type { ModelRankingMetric, ModelRankingPeriod, ModelRankingResponse } from '../types';

export const MODEL_RANKING_POLL_INTERVAL_MS = 60_000;

export interface ModelRankingDataAPI {
  leaderboard: (
    period: ModelRankingPeriod,
    metric: ModelRankingMetric,
    signal?: AbortSignal,
  ) => Promise<ModelRankingResponse>;
}

export interface UseModelRankingDataOptions {
  enabled: boolean;
  period: ModelRankingPeriod;
  metric: ModelRankingMetric;
  onAuthRequired?: () => void;
  onBackgroundRefreshError?: (error: unknown) => void;
  api?: ModelRankingDataAPI;
}

const defaultAPI: ModelRankingDataAPI = { leaderboard: fetchModelRankingLeaderboard };
const isAbortError = (error: unknown) => error instanceof DOMException && error.name === 'AbortError';
const leaderboardKey = (period: ModelRankingPeriod, metric: ModelRankingMetric) => `${period}:${metric}`;

// useModelRankingData maintains the read-only model leaderboard only. It aborts
// stale requests, caches per period:metric, keeps the last board visible when a
// silent refresh fails, and polls every 60 s only while the document is visible.
export function useModelRankingData({
  enabled,
  period,
  metric,
  onAuthRequired,
  onBackgroundRefreshError,
  api = defaultAPI,
}: UseModelRankingDataOptions) {
  const [leaderboard, setLeaderboard] = useState<ModelRankingResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const generationRef = useRef(0);
  const cacheRef = useRef(new Map<string, ModelRankingResponse>());

  const loadLeaderboard = useCallback(async (
    nextPeriod: ModelRankingPeriod,
    nextMetric: ModelRankingMetric,
    { silent = false }: { silent?: boolean } = {},
  ) => {
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    const generation = ++generationRef.current;
    const key = leaderboardKey(nextPeriod, nextMetric);
    const cachedBoard = cacheRef.current.get(key) ?? null;
    if (!silent) {
      setLoading(true);
      setError(null);
      // Keep the previous response while the target period/metric loads so the
      // card does not flash; the content still belongs to the new selection.
      setLeaderboard((current) => cachedBoard ?? current);
    }
    try {
      const board = await api.leaderboard(nextPeriod, nextMetric, controller.signal);
      if (generationRef.current === generation) {
        cacheRef.current.set(key, board);
        setLeaderboard(board);
        setError(null);
      }
      return board;
    } catch (loadError) {
      if (!isAbortError(loadError) && generationRef.current === generation) {
        if (loadError instanceof ModelRankingApiError && loadError.status === 401) onAuthRequired?.();
        if (cachedBoard) {
          setLeaderboard({ ...cachedBoard, stale: true });
          setError(null);
          onBackgroundRefreshError?.(loadError);
        } else {
          setLeaderboard(null);
          setError(loadError);
        }
      }
      return null;
    } finally {
      if (controllerRef.current === controller) controllerRef.current = null;
      if (generationRef.current === generation) setLoading(false);
    }
  }, [api, onAuthRequired, onBackgroundRefreshError]);

  const refresh = useCallback(
    () => loadLeaderboard(period, metric),
    [loadLeaderboard, metric, period],
  );

  useEffect(() => {
    if (!enabled) {
      controllerRef.current?.abort();
      controllerRef.current = null;
      setLoading(false);
      return;
    }
    void loadLeaderboard(period, metric);
    return () => controllerRef.current?.abort();
  }, [enabled, loadLeaderboard, metric, period]);

  useEffect(() => {
    if (!enabled) return;
    const interval = window.setInterval(() => {
      if (document.visibilityState === 'hidden') return;
      void loadLeaderboard(period, metric, { silent: true });
    }, MODEL_RANKING_POLL_INTERVAL_MS);
    return () => window.clearInterval(interval);
  }, [enabled, loadLeaderboard, metric, period]);

  useEffect(() => () => controllerRef.current?.abort(), []);

  return {
    leaderboard,
    loading,
    error,
    refresh,
  };
}