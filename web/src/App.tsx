import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import './index.css';
import './App.css';
import './embed/cpamcEmbed.css';
import { ApiError, appPath, clearEmbedSessionToken, getSession, login, loginWithCPAAPIKey, startOIDCLogin } from './lib/api';
import type { AuthRole, AuthSessionAPIKeySummary } from './lib/types';
import { AppFooter } from './components/AppFooter';
import { isKeyViewerPath, type KeyViewerPath } from './features/key-viewer';
import { KeyAnalysisPage } from './pages/KeyAnalysisPage';
import { KeyEventsPage } from './pages/KeyEventsPage';
import { KeyLiveFlowPage } from './pages/KeyLiveFlowPage';
import { KeyOverviewPage } from './pages/KeyOverviewPage';
import { KeyRankingPage } from './pages/KeyRankingPage';
import { LoginPage } from './pages/LoginPage';
import { UsagePage } from './pages/UsagePage';
import { ProviderDetailPage } from './pages/ProviderDetailPage';
import { cpamcEmbedSearch, isCPAMCEmbed, notifyCPAMCEmbedReady } from './embed/cpamcEmbed';
import { getUsageTabPath, resolveProviderDetailIdFromPath, resolveUsageTabFromPath, stripAppBasePath } from './lib/usageNavigation';
import { useUsageStatsStore } from './stores/useUsageStatsStore';

type AuthState = 'checking' | 'authenticated' | 'unauthenticated';
const getInitialKeyViewerPath = (): KeyViewerPath => {
  if (typeof window === 'undefined') return '/key-overview';
  const currentPath = stripAppBasePath(window.location.pathname, window.__APP_BASE_PATH__) ?? '/';
  return isKeyViewerPath(currentPath) ? currentPath : '/key-overview';
};

export const getRoleHomePath = (role: AuthRole): '/' | '/key-overview' => (
  role === 'api_key_viewer' ? '/key-overview' : '/'
);

export const getRoleTargetPath = (
  role: AuthRole,
  currentPath: string,
  isEmbeddedInCPAMC = false,
): string => {
  // 路径白名单与会话角色共同决定落点；未知路径只回到该角色自己的首页。
  if (role === 'api_key_viewer') {
    return isKeyViewerPath(currentPath) ? currentPath : '/key-overview';
  }
  if (currentPath === '/') return '/';
  if (resolveProviderDetailIdFromPath(currentPath)) return currentPath;

  const usageTab = resolveUsageTabFromPath(currentPath);
  if (!usageTab || (isEmbeddedInCPAMC && (usageTab === 'ranking' || usageTab === 'live-flow'))) return '/';
  return getUsageTabPath(usageTab);
};

export const shouldNormalizeRolePath = (
  role: AuthRole,
  currentPath: string,
  isEmbeddedInCPAMC = false,
): boolean => currentPath !== getRoleTargetPath(role, currentPath, isEmbeddedInCPAMC);

export const getOIDCLoginErrorKey = (code: string): string => {
  const allowed = ['oidc_canceled', 'oidc_expired', 'oidc_denied', 'oidc_unavailable', 'oidc_public_origin', 'login_rate_limited'];
  return allowed.includes(code) ? `auth.${code}` : 'auth.oidc_failed';
};

function App() {
  const { t } = useTranslation();
  const [authState, setAuthState] = useState<AuthState>('checking');
  const [authRole, setAuthRole] = useState<AuthRole | null>(null);
  const [sessionAPIKey, setSessionAPIKey] = useState<AuthSessionAPIKeySummary | undefined>();
  const [keyViewerPath, setKeyViewerPath] = useState<KeyViewerPath>(getInitialKeyViewerPath);
  const [adminLoginErrorKey, setAdminLoginErrorKey] = useState('');
  const [apiKeyLoginError, setAPIKeyLoginError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [oidcLoginEnabled, setOIDCLoginEnabled] = useState(false);
  const clearUsageStats = useUsageStatsStore((state) => state.clearUsageStats);
  const isEmbeddedInCPAMC = isCPAMCEmbed();

  const clearSession = useCallback(() => {
    clearEmbedSessionToken();
    clearUsageStats();
    setAuthState('unauthenticated');
    setAuthRole(null);
    setSessionAPIKey(undefined);
  }, [clearUsageStats]);

  const applySession = useCallback((session: Awaited<ReturnType<typeof getSession>>) => {
    setOIDCLoginEnabled(Boolean(session.oidc_login_enabled));
    if (!session.authenticated) {
      clearSession();
      return;
    }
    setAuthState('authenticated');
    setAuthRole(session.role ?? 'admin');
    setSessionAPIKey(session.api_key);
  }, [clearSession]);

  const loadSession = useCallback(async () => {
    const session = await getSession();
    applySession(session);
    return session;
  }, [applySession]);

  useEffect(() => {
    void loadSession().catch(() => {
      clearSession();
    });
  }, [clearSession, loadSession]);

  useEffect(() => {
    notifyCPAMCEmbedReady();
  }, []);

  useEffect(() => {
    const current = new URL(window.location.href);
    const errorCode = current.searchParams.get('oidc_error');
    if (!errorCode) return;
    // 回调只传本地错误枚举；消费后立即移除，刷新不会重复显示，也不展示 IdP 原文。
    setAdminLoginErrorKey(getOIDCLoginErrorKey(errorCode));
    current.searchParams.delete('oidc_error');
    window.history.replaceState(null, '', current.pathname + current.search + current.hash);
  }, []);

  useEffect(() => {
    if (authState !== 'authenticated' || !authRole) return;
    const strippedPath = stripAppBasePath(window.location.pathname, window.__APP_BASE_PATH__);
    const targetPath = getRoleTargetPath(authRole, strippedPath ?? '/', isEmbeddedInCPAMC);
    if (authRole === 'api_key_viewer') {
      setKeyViewerPath(targetPath as KeyViewerPath);
    }
    if (strippedPath === targetPath) return;
    window.history.replaceState(null, '', appPath(targetPath) + cpamcEmbedSearch());
  }, [authRole, authState, isEmbeddedInCPAMC]);

  const handlePasswordLogin = useCallback(async (password: string) => {
    setSubmitting(true);
    setAdminLoginErrorKey('');
    try {
      await login(password);
      const session = await loadSession();
      if (!session.authenticated) {
        setAdminLoginErrorKey('auth.login_failed');
        clearSession();
        return;
      }
      const currentPath = stripAppBasePath(window.location.pathname, window.__APP_BASE_PATH__) ?? '/';
      const targetPath = getRoleTargetPath(session.role ?? 'admin', currentPath, isEmbeddedInCPAMC);
      // 管理员凭据直达链接需跨登录保留；发生角色或路由重定向时仍只携带嵌入标记。
      const keepCredentialSearch = targetPath === currentPath && (
        targetPath === '/auth-files'
        || targetPath === '/credentials'
        || targetPath === '/ai-provider'
        || resolveProviderDetailIdFromPath(targetPath) !== null
      );
      const suffix = keepCredentialSearch ? window.location.search + window.location.hash : cpamcEmbedSearch();
      window.history.replaceState(null, '', appPath(targetPath) + suffix);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setAdminLoginErrorKey('auth.invalid_password');
      } else {
        setAdminLoginErrorKey('auth.login_failed');
      }
      clearSession();
    } finally {
      setSubmitting(false);
    }
  }, [clearSession, isEmbeddedInCPAMC, loadSession]);

  const handleAPIKeyLogin = useCallback(async (apiKey: string) => {
    setSubmitting(true);
    setAPIKeyLoginError('');
    try {
      await loginWithCPAAPIKey(apiKey);
      const session = await loadSession();
      if (!session.authenticated || session.role !== 'api_key_viewer') {
        setAPIKeyLoginError(t('auth.api_key_login_failed'));
        clearSession();
        return;
      }
      const currentPath = stripAppBasePath(window.location.pathname, window.__APP_BASE_PATH__) ?? '/';
      const targetPath = getRoleTargetPath(session.role, currentPath, isEmbeddedInCPAMC) as KeyViewerPath;
      setKeyViewerPath(targetPath);
      window.history.replaceState(null, '', appPath(targetPath) + cpamcEmbedSearch());
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setAPIKeyLoginError(t('auth.invalid_api_key'));
      } else if (error instanceof ApiError && error.status === 429) {
        setAPIKeyLoginError(t('auth.login_rate_limited'));
      } else {
        setAPIKeyLoginError(t('auth.api_key_login_failed'));
      }
      clearSession();
    } finally {
      setSubmitting(false);
    }
  }, [clearSession, isEmbeddedInCPAMC, loadSession, t]);

  const handleOIDCLogin = useCallback(async () => {
    setSubmitting(true);
    setAdminLoginErrorKey('');
    try {
      const target = window.location.pathname + window.location.search + window.location.hash;
      const { authorization_url: authorizationURL } = await startOIDCLogin(target);
      window.location.assign(authorizationURL);
    } catch (error) {
      const code = error instanceof ApiError && error.status === 429
        ? 'login_rate_limited'
        : error instanceof ApiError ? error.code ?? 'oidc_failed' : 'oidc_unavailable';
      setAdminLoginErrorKey(getOIDCLoginErrorKey(code));
      setSubmitting(false);
    }
  }, []);

  const handleKeyViewerNavigate = useCallback((path: KeyViewerPath) => {
    if (path === keyViewerPath) return;
    window.history.replaceState(null, '', appPath(path) + cpamcEmbedSearch());
    setKeyViewerPath(path);
  }, [keyViewerPath]);

  let page: ReactNode;
  if (authState === 'checking') {
    page = <div className="app-checking" aria-busy="true" />;
  } else if (authState === 'unauthenticated') {
    page = <LoginPage loading={submitting} oidcEnabled={oidcLoginEnabled && !isEmbeddedInCPAMC} onOIDCLogin={handleOIDCLogin} adminError={adminLoginErrorKey ? t(adminLoginErrorKey) : ''} apiKeyError={apiKeyLoginError} onPasswordSubmit={handlePasswordLogin} onAPIKeySubmit={handleAPIKeyLogin} />;
  } else if (authRole === 'api_key_viewer') {
    const commonProps = {
      apiKey: sessionAPIKey,
      onNavigate: handleKeyViewerNavigate,
      onAuthRequired: clearSession,
    };

    switch (keyViewerPath) {
      case '/key-analysis':
        page = <KeyAnalysisPage {...commonProps} />;
        break;

      case '/key-ranking':
        page = <KeyRankingPage {...commonProps} />;
        break;

      case '/key-events':
        page = <KeyEventsPage {...commonProps} />;
        break;

      case '/key-live-flow':
        page = <KeyLiveFlowPage {...commonProps} />;
        break;

      case '/key-realtime':
        page = <KeyOverviewPage {...commonProps} page="realtime" />;
        break;

      default:
        page = <KeyOverviewPage {...commonProps} page="overview" />;
        break;
    }
  } else {
    const currentPath = stripAppBasePath(window.location.pathname, window.__APP_BASE_PATH__) ?? '/';
    const providerId = resolveProviderDetailIdFromPath(currentPath);
    page = providerId
      ? <ProviderDetailPage providerId={providerId} onAuthRequired={clearSession} />
      : <UsagePage onAuthRequired={clearSession} />;
  }

  return (
    <div className="app-frame" data-embed={isEmbeddedInCPAMC ? 'cpamc' : undefined}>
      <main className="app-main">{page}</main>
      <AppFooter loadVersion={authState === 'authenticated'} />
    </div>
  );
}

export default App;
