import React, { memo, useState } from 'react';
import { Key, Plus, Trash2, Bell, Globe, Shield, Monitor, Sparkles, ScanLine, Settings2 } from 'lucide-react';
import { useStore } from '../store/useStore';
import { t } from '../i18n';
import { SkeletonLoader } from './SkeletonLoader';
import { InlineError } from './ui/InlineError';
import { GlassPanel } from './ui/GlassPanel';

interface ApiKey {
  id: string;
  label: string;
  key: string;
  createdAt: number;
}

interface NotificationChannel {
  id: string;
  name: string;
  type: string;
  configJson: string;
}

interface SettingsViewProps {
  apiKeys: ApiKey[];
  notificationChannels: NotificationChannel[];
  onCreateApiKey: (label: string) => void;
  onDeleteApiKey: (id: string) => void;
  onSendWebhook: (url: string, payloadJson: string, secret: string) => void;
  inline?: boolean;
  isLoading?: boolean;
  error?: string | null;
}

const SettingsViewComponent: React.FC<SettingsViewProps> = ({
   apiKeys, notificationChannels, onCreateApiKey, onDeleteApiKey, onSendWebhook, inline = false, isLoading, error
 }) => {
    const enableScanline = useStore(s => s.enableScanline)
    const enableGlow = useStore(s => s.enableGlow)
    const enableFlicker = useStore(s => s.enableFlicker)
    const expandedSections = useStore(s => s.expandedSections)
    const toggleSection = useStore(s => s.toggleSection)
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [webhookUrl, setWebhookUrl] = useState('');

  const [webhookPayload, setWebhookPayload] = useState('{}');
  const [webhookSecret, setWebhookSecret] = useState('');
  const [newKeyLabel, setNewKeyLabel] = useState('');

  if (isLoading) return <SkeletonLoader />;
  if (error) return <div className="max-w-4xl mx-auto"><InlineError message={error} /></div>;

  return (
    <div 
      className={(inline ? '' : 'max-w-4xl mx-auto ') + 'space-y-8'} 
      role="region" 
      aria-label="Settings"
    >
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <h2 className="text-3xl font-bold tracking-tight">{t('settings.title')}</h2>
          <p className="text-textMuted text-sm mt-1">Gestisci chiavi API, notifiche e integrazioni.</p>
        </div>
        <button
          onClick={() => setShowAdvanced(v => !v)}
          className={`p-2 rounded-lg transition-colors ${showAdvanced ? 'bg-primary/10 text-primary' : 'text-textMuted hover:text-text hover:bg-surface-alt'}`}
          aria-label="Toggle advanced settings"
          title="Advanced settings"
        >
          <Settings2 size={20} />
        </button>
      </div>

      <GlassPanel
        header="Quick Settings"
        icon={<Monitor size={16} />}
        sectionKey="settings.quick"
        expanded={expandedSections['settings.quick']}
        onToggle={() => toggleSection('settings.quick')}
      >
        <div className="space-y-3">
          <div className="flex items-center justify-between p-4 bg-surface-alt rounded-2xl">
            <div className="flex items-center space-x-3">
              <ScanLine size={16} className="text-primary" />
              <span className="font-bold text-sm">{t('settings.scanlines')}</span>
            </div>
            <button 
              onClick={() => useStore.getState().setEnableScanline(!enableScanline)}
              className={`w-10 h-5 rounded-full relative transition-colors focus:ring-2 focus:ring-primary ${enableScanline ? 'bg-primary' : 'bg-border'}`}
              aria-label="Toggle scanlines"
              role="switch"
              aria-checked={enableScanline}
            >
              <div className={`absolute top-1 w-3 h-3 rounded-full bg-white transition-all ${enableScanline ? 'left-6' : 'left-1'}`} />
            </button>
          </div>
          <div className="flex items-center justify-between p-4 bg-surface-alt rounded-2xl">
            <div className="flex items-center space-x-3">
              <Sparkles size={16} className="text-primary" />
              <span className="font-bold text-sm">{t('settings.glow')}</span>
            </div>
            <button 
              onClick={() => useStore.getState().setEnableGlow(!enableGlow)}
              className={`w-10 h-5 rounded-full relative transition-colors focus:ring-2 focus:ring-primary ${enableGlow ? 'bg-primary' : 'bg-border'}`}
              aria-label="Toggle glow effect"
              role="switch"
              aria-checked={enableGlow}
            >
              <div className={`absolute top-1 w-3 h-3 rounded-full bg-white transition-all ${enableGlow ? 'left-6' : 'left-1'}`} />
            </button>
          </div>
          <div className="flex items-center justify-between p-4 bg-surface-alt rounded-2xl">
            <div className="flex items-center space-x-3">
              <Monitor size={16} className="text-primary" />
              <span className="font-bold text-sm">{t('settings.flicker')}</span>
            </div>
            <button 
              onClick={() => useStore.getState().setEnableFlicker(!enableFlicker)}
              className={`w-10 h-5 rounded-full relative transition-colors focus:ring-2 focus:ring-primary ${enableFlicker ? 'bg-primary' : 'bg-border'}`}
              aria-label="Toggle flicker effect"
              role="switch"
              aria-checked={enableFlicker}
            >
              <div className={`absolute top-1 w-3 h-3 rounded-full bg-white transition-all ${enableFlicker ? 'left-6' : 'left-1'}`} />
            </button>
          </div>

          {apiKeys.length > 0 && (
            <div className="p-4 bg-surface-alt rounded-2xl flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <Key size={16} className="text-textMuted" />
                <span className="text-sm text-textMuted">{apiKeys[0].label}: {apiKeys[0].key.slice(0, 4)}••••••</span>
              </div>
            </div>
          )}
        </div>
      </GlassPanel>

      <GlassPanel
        header="All Settings"
        icon={<Key size={16} />}
        sectionKey="settings.all"
        expanded={expandedSections['settings.all']}
        onToggle={() => toggleSection('settings.all')}
      >
        <div className="bg-surface rounded-lg border border-border shadow-sm overflow-hidden">
          <div className="p-6 border-b border-border">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-primary/10 rounded-xl flex items-center justify-center text-primary"><Key size={20} /></div>
              <div>
                <h3 className="font-bold text-lg">{t('settings.apiKey')}</h3>
                <p className="text-[10px] text-textMuted uppercase tracking-widest font-bold">Gestione chiavi di accesso</p>
              </div>
            </div>
          </div>
          <div className="p-6 space-y-4">
            {apiKeys.map(k => (
              <div key={k.id} className="flex items-center justify-between px-8 py-4 hover:bg-surface-alt/50 transition-colors group">
                <div className="flex items-center space-x-4">
                  <Shield size={16} className="text-textMuted" />
                  <div>
                    <span className="font-bold text-sm">{k.label}</span>
                    <span className="text-[10px] font-mono text-textMuted ml-3">{k.key ? k.key.slice(0, 4) + '••••••' : '••••••••'}</span>
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  <span className="text-[10px] text-textMuted">{new Date(k.createdAt * 1000).toLocaleDateString()}</span>
                  <button
                    onClick={() => { if (confirm('Revocare questa chiave?')) onDeleteApiKey(k.id); }}
                    className="p-1.5 text-textDim hover:text-danger hover:bg-danger/10 rounded-lg transition-all opacity-0 group-hover:opacity-100 focus:ring-2 focus:ring-primary"
                    aria-label={t('settings.revoke')}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            ))}
            {apiKeys.length === 0 && (
              <div className="px-8 py-12 text-center">
                <Key size={32} className="mx-auto text-textDim mb-3" />
                <p className="text-textMuted text-xs font-bold uppercase tracking-widest">Nessuna chiave API configurata</p>
              </div>
            )}
          </div>
        </div>

        <div className="bg-surface rounded-lg border border-border shadow-sm overflow-hidden mt-4">
          <div className="p-6 border-b border-border">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-warning/10 rounded-xl flex items-center justify-center text-warning"><Bell size={20} /></div>
              <div>
                <h3 className="font-bold text-lg">Canali di Notifica</h3>
                <p className="text-[10px] text-textMuted uppercase tracking-widest font-bold">Webhook e integrazioni di notifica</p>
              </div>
            </div>
          </div>
          <div className="p-6 space-y-6">
            {notificationChannels.map(ch => (
              <div key={ch.id} className="flex items-center justify-between p-4 bg-surface-alt rounded-2xl">
                <div className="flex items-center space-x-3">
                  <Globe size={16} className="text-warning" />
                  <div>
                    <span className="font-bold text-sm">{ch.name}</span>
                    <span className="text-[10px] bg-warning/10 text-warning px-2 py-0.5 rounded ml-2 uppercase font-bold">{ch.type}</span>
                  </div>
                </div>
              </div>
            ))}
            {notificationChannels.length === 0 && (
              <div className="py-8 text-center">
                <Bell size={32} className="mx-auto text-textDim mb-3" />
                <p className="text-textMuted text-xs font-bold uppercase tracking-widest">Nessun canale configurato</p>
              </div>
            )}

            <div className="border-t border-border pt-6 space-y-4">
              <div className="text-[10px] font-black text-textMuted uppercase tracking-widest">Test Webhook</div>
              <input
                value={webhookUrl}
                onChange={(e) => setWebhookUrl(e.target.value)}
                placeholder="https://hooks.example.com/..."
                className="w-full px-4 py-3 border border-border rounded-xl text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/10 bg-surface text-text"
              />
              <textarea
                value={webhookPayload}
                onChange={(e) => setWebhookPayload(e.target.value)}
                rows={3}
                className="w-full px-4 py-3 border border-border rounded-xl text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/10 resize-none bg-surface text-text"
                placeholder='{"event": "test"}'
              />
              <input
                value={webhookSecret}
                onChange={(e) => setWebhookSecret(e.target.value)}
                placeholder={t('settings.webhookSecret')}
                className="w-full px-4 py-3 border border-border rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-primary/10 bg-surface text-text"
              />
              <button
                onClick={() => onSendWebhook(webhookUrl, webhookPayload, webhookSecret)}
                disabled={!webhookUrl.trim()}
                className="px-6 py-3 bg-warning text-white rounded-xl text-sm font-bold hover:bg-warning/90 disabled:opacity-50 transition-all focus:ring-2 focus:ring-primary"
                aria-label="Send test webhook"
              >
                {t('settings.sendTest')}
              </button>
            </div>
          </div>
        </div>
      </GlassPanel>

      
      <GlassPanel
        header="Developer"
        icon={<Settings2 size={16} />}
        sectionKey="settings.advanced"
        expanded={expandedSections['settings.advanced']}
        onToggle={() => toggleSection('settings.advanced')}
        advanced
        showAdvanced={showAdvanced}
      >
        <div className="space-y-4 p-4 bg-surface-alt rounded-lg">
          <div className="flex items-center justify-between">
            <span className="text-sm font-semibold">Debug Logging</span>
            <span className="text-[10px] text-textMuted font-mono">INFO</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm font-semibold">Feature Flags</span>
            <span className="text-[10px] text-textMuted">genesis, sandbox, mcp</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm font-semibold">DuckDB Query Inspector</span>
            <span className="text-[10px] text-textMuted">—</span>
          </div>
        </div>
      </GlassPanel>
    </div>
  );
};

export const SettingsView = memo(SettingsViewComponent);