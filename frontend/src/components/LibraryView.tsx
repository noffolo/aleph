import React from 'react';
import { Book, FileText, Download, Trash2, Upload } from 'lucide-react';
import { useStore } from '../store/useStore';
import { t } from '../i18n';
import { SkeletonLoader } from './SkeletonLoader';
import { InlineError } from './ui/InlineError';
import { GlassPanel } from './ui/GlassPanel';

interface Asset {
  id: string;
  name: string;
  type: string;
  createdAt: number;
}

interface LibraryViewProps {
  assets: Asset[];
  onViewAsset: (id: string) => void;
  onDeleteAsset: (id: string) => void;
  selectedAssetContent: string | null;
  setSelectedAssetContent: (val: string | null) => void;
  selectedAssetName?: string;
  onGetAssetContent: (id: string) => Promise<string>;
  onGeneratePdf: (id: string) => Promise<{ pdfData: Uint8Array; filename: string }>;
  onUploadAsset: (filename: string, content: Uint8Array) => Promise<void>;
  selectedAssetId?: string | null;
  inline?: boolean;
  isLoading?: boolean;
  error?: string | null;
}

export const LibraryView: React.FC<LibraryViewProps> = React.memo(({ assets, onViewAsset, onDeleteAsset, selectedAssetContent, onGetAssetContent, onGeneratePdf, onUploadAsset, selectedAssetId, inline = false, isLoading, error }) => {
  const [uploading, setUploading] = React.useState(false);
  const [dragOver, setDragOver] = React.useState(false);
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const expandedSections = useStore(s => s.expandedSections);
  const toggleSection = useStore(s => s.toggleSection);

  if (isLoading) return <SkeletonLoader />;
  if (error) return <div className="max-w-6xl mx-auto"><InlineError message={error} /></div>;

  const handleDownload = async (asset: Asset) => {
    let content = selectedAssetContent || '';
    if (!content) {
      content = await onGetAssetContent(asset.id);
    }
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = asset.name || 'asset.txt';
    a.click();
    URL.revokeObjectURL(url);
  };

  const processFiles = async (files: FileList) => {
    setUploading(true);
    try {
      for (const file of Array.from(files)) {
        const buffer = await file.arrayBuffer();
        await onUploadAsset(file.name, new Uint8Array(buffer));
      }
    } finally {
      setUploading(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    if (e.dataTransfer.files.length > 0) {
      processFiles(e.dataTransfer.files);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      processFiles(e.target.files);
      e.target.value = '';
    }
  };

  const openAssetDetail = (id: string) => {
    useStore.getState().setSlideOverContent({ type: 'asset', title: 'Dettaglio Asset', data: { assetId: id } });
  };

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-8'}>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">{t('library.title')}</h2>
          <p className="text-textMuted text-sm mt-1">{t('library.subtitle')}</p>
        </div>
        <button
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          className="flex items-center space-x-2 bg-primary text-white px-6 py-3 rounded-lg font-bold hover:bg-primary/90 transition-all shadow-lg  disabled:opacity-50"
        >
          <Upload size={20} />
          <span>{uploading ? t('generic.loadingLower') : t('library.upload')}</span>
        </button>
        <input ref={fileInputRef} type="file" multiple onChange={handleFileSelect} className="hidden" />
      </div>

      <div
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        className={`rounded-lg border-2 border-dashed transition-all p-2 ${dragOver ? 'border-primary bg-primary/5' : 'border-transparent'}`}
      >
        <GlassPanel
          header="Library"
          sectionKey="library.list"
          icon={<Book size={16} />}
          expanded={!!expandedSections['library.list']}
          onToggle={() => toggleSection('library.list')}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {assets.map(a => (
            <div key={a.id} className="bg-surface p-6 rounded-lg border border-border shadow-sm hover:shadow-lg transition-all group relative">
<button
  onClick={(e) => { 
    e.stopPropagation()
    useStore.getState().setSlideOverContent({ 
      type: 'confirm', 
      title: 'Conferma eliminazione', 
      data: { message: 'Sei sicuro di voler eliminare questo asset?', onConfirm: () => onDeleteAsset(a.id) } 
    })
  }}
  className="absolute top-6 right-6 p-2 text-textDim hover:text-danger hover:bg-danger/10 rounded-xl transition-all opacity-0 group-hover:opacity-100"
  aria-label={`Elimina ${a.name}`}
>
  <Trash2 size={18} />
</button>
               <div className="w-12 h-12 bg-primary/10 rounded-lg flex items-center justify-center text-primary mb-4 group-hover:bg-primary group-hover:text-white transition-colors">
                  <FileText size={24} />
               </div>
               <h3 className="text-xl font-bold mb-1 truncate">{a.name}</h3>
               <div className="flex items-center space-x-2 text-[10px] text-textMuted font-bold uppercase tracking-widest mb-6">
                  <span>{new Date(a.createdAt * 1000).toLocaleDateString()}</span>
                  <span className="bg-surface-alt px-2 py-0.5 rounded text-textMuted">{a.type}</span>
               </div>
               <div className="flex items-center space-x-2">
                  <button
                     onClick={() => openAssetDetail(a.id)}
                      className="flex-1 py-3 bg-surface-alt text-text rounded-xl text-xs font-bold hover:bg-danger/20 transition-colors flex items-center justify-center space-x-2"
                  >
                     <span>Leggi Report</span>
                  </button>
<button
  onClick={(e) => { e.stopPropagation(); handleDownload(a); }}
  className="p-3 bg-surface-alt text-textMuted rounded-xl hover:bg-surface-alt hover:text-text transition-all"
  aria-label={`Scarica ${a.name}`}
>
  <Download size={16} />
</button>
               </div>
            </div>
          ))}
          </div>
        </GlassPanel>
        {assets.length === 0 && !dragOver && (
          <div className="col-span-full py-24 bg-surface border-2 border-dashed border-border rounded-lg text-center">
             <Book size={48} className="mx-auto text-textDim mb-4" />
              <p className="text-textMuted font-bold uppercase text-[10px] tracking-[0.2em]">Nessun report generato in questo spazio di lavoro</p>
               <p className="text-textDim text-xs mt-2">{t('library.dragAndDrop')}</p>
          </div>
        )}
        {dragOver && (
          <div className="col-span-full py-24 border-2 border-primary bg-primary/5 rounded-lg text-center">
            <Upload size={48} className="mx-auto text-primary/70 mb-4" />
            <p className="text-primary font-bold text-sm">{t('library.dropToUpload')}</p>
          </div>
        )}
      </div>
    </div>
  );
});
