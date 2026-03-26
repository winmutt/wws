import React, { useState, useEffect } from 'react';
import { workspaces, WorkspaceExport, WorkspaceImport } from '../services/api';

interface WorkspaceExportModalProps {
  workspaceId: number;
  workspaceName: string;
  isOpen: boolean;
  onClose: () => void;
  onExportComplete?: () => void;
}

interface WorkspaceImportModalProps {
  organizationId: number;
  isOpen: boolean;
  onClose: () => void;
  onImportComplete?: () => void;
}

const WorkspaceExportModal: React.FC<WorkspaceExportModalProps> = ({
  workspaceId,
  workspaceName,
  isOpen,
  onClose,
  onExportComplete,
}) => {
  const [format, setFormat] = useState<'json' | 'tar' | 'zip'>('json');
  const [includeData, setIncludeData] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [exportStatus, setExportStatus] = useState<WorkspaceExport | null>(null);
  const [error, setError] = useState('');
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    if (isOpen) {
      // Reset state when modal opens
      setExportStatus(null);
      setError('');
      setProgress(0);
    }
  }, [isOpen]);

  const handleExport = async () => {
    setIsExporting(true);
    setError('');
    setProgress(0);

    try {
      // Start export
      const exportResult = await workspaces.exportWorkspace(workspaceId, {
        format,
        include_data: includeData,
      });

      setExportStatus(exportResult);

      // Poll for completion
      let attempts = 0;
      const maxAttempts = 30;
      
      while (attempts < maxAttempts && exportResult.status !== 'completed') {
        await new Promise(resolve => setTimeout(resolve, 1000));
        const status = await workspaces.getExportStatus(exportResult.id);
        setExportStatus(status);
        setProgress(Math.min(90, ((attempts + 1) / maxAttempts) * 100));
        
        if (status.status === 'failed') {
          throw new Error(status.error_message || 'Export failed');
        }
        
        attempts++;
      }

      setProgress(100);

      if (onExportComplete) {
        onExportComplete();
      }
    } catch (err) {
      console.error('Export failed:', err);
      setError(err instanceof Error ? err.message : 'Failed to export workspace');
      setProgress(0);
    } finally {
      setIsExporting(false);
    }
  };

  const handleDownload = async () => {
    if (!exportStatus) return;

    try {
      const blob = await workspaces.downloadExport(exportStatus.id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `workspace-${workspaceName}-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      console.error('Download failed:', err);
      setError('Failed to download export');
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="p-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold text-gray-900">Export Workspace</h2>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
              disabled={isExporting}
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="mb-4">
            <p className="text-sm text-gray-600 mb-4">
              Export configuration for workspace: <strong>{workspaceName}</strong>
            </p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Export Format
                </label>
                <select
                  value={format}
                  onChange={(e) => setFormat(e.target.value as 'json' | 'tar' | 'zip')}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  disabled={isExporting}
                >
                  <option value="json">JSON</option>
                  <option value="tar">TAR (Coming soon)</option>
                  <option value="zip">ZIP (Coming soon)</option>
                </select>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="includeData"
                  checked={includeData}
                  onChange={(e) => setIncludeData(e.target.checked)}
                  className="mr-2"
                  disabled={isExporting}
                />
                <label htmlFor="includeData" className="text-sm text-gray-700">
                  Include workspace data files
                  <span className="block text-xs text-gray-500">
                    Note: Data export not yet implemented
                  </span>
                </label>
              </div>
            </div>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {exportStatus && exportStatus.status === 'completed' && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg">
              <p className="text-sm text-green-700">
                Export completed successfully!
                {exportStatus.file_size_mb && (
                  <span> Size: {exportStatus.file_size_mb.toFixed(2)} MB</span>
                )}
              </p>
              <p className="text-xs text-gray-600 mt-1">
                Expires: {new Date(exportStatus.expires_at).toLocaleString()}
              </p>
            </div>
          )}

          {isExporting && (
            <div className="mb-4">
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <p className="text-sm text-gray-600 mt-2">Exporting... {Math.round(progress)}%</p>
            </div>
          )}

          <div className="flex space-x-3 mt-6">
            {exportStatus && exportStatus.status === 'completed' ? (
              <>
                <button
                  onClick={handleDownload}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                >
                  Download
                </button>
                <button
                  onClick={onClose}
                  className="flex-1 bg-gray-200 hover:bg-gray-300 text-gray-800 px-4 py-2 rounded-lg font-medium transition-colors"
                >
                  Close
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={handleExport}
                  disabled={isExporting}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                >
                  {isExporting ? 'Exporting...' : 'Export'}
                </button>
                <button
                  onClick={onClose}
                  disabled={isExporting}
                  className="flex-1 bg-gray-200 hover:bg-gray-300 disabled:bg-gray-100 text-gray-800 px-4 py-2 rounded-lg font-medium transition-colors"
                >
                  Cancel
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

const WorkspaceImportModal: React.FC<WorkspaceImportModalProps> = ({
  organizationId,
  isOpen,
  onClose,
  onImportComplete,
}) => {
  const [importFile, setImportFile] = useState<File | null>(null);
  const [workspaceName, setWorkspaceName] = useState('');
  const [isImporting, setIsImporting] = useState(false);
  const [importStatus, setImportStatus] = useState<WorkspaceImport | null>(null);
  const [error, setError] = useState('');
  const [progress, setProgress] = useState(0);
  const [importHistory, setImportHistory] = useState<WorkspaceImport[]>([]);

  useEffect(() => {
    if (isOpen) {
      loadImportHistory();
      setImportFile(null);
      setWorkspaceName('');
      setImportStatus(null);
      setError('');
      setProgress(0);
    }
  }, [isOpen, organizationId]);

  const loadImportHistory = async () => {
    try {
      const history = await workspaces.listImports(organizationId);
      setImportHistory(history);
    } catch (err) {
      console.error('Failed to load import history:', err);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setImportFile(e.target.files[0]);
      // Auto-suggest name from filename
      const name = e.target.files[0].name.replace(/\.[^/.]+$/, '');
      setWorkspaceName(name);
    }
  };

  const handleImport = async () => {
    if (!importFile) {
      setError('Please select a file to import');
      return;
    }

    setIsImporting(true);
    setError('');
    setProgress(0);

    try {
      // Note: For now, we're using a placeholder export path
      // In a real implementation, we'd upload the file first
      const importResult = await workspaces.importWorkspace({
        export_path: importFile.name,
        format: importFile.name.endsWith('.json') ? 'json' : 'unknown',
        name: workspaceName || `imported-${new Date().toISOString().split('T')[0]}`,
        organization_id: organizationId,
      });

      setImportStatus(importResult);

      // Poll for completion
      let attempts = 0;
      const maxAttempts = 30;
      
      while (attempts < maxAttempts && importResult.status !== 'completed') {
        await new Promise(resolve => setTimeout(resolve, 1000));
        const status = await workspaces.getImportStatus(importResult.id);
        setImportStatus(status);
        setProgress(Math.min(90, ((attempts + 1) / maxAttempts) * 100));
        
        if (status.status === 'failed') {
          throw new Error(status.error || 'Import failed');
        }
        
        attempts++;
      }

      setProgress(100);
      loadImportHistory();

      if (onImportComplete) {
        onImportComplete();
      }
    } catch (err) {
      console.error('Import failed:', err);
      setError(err instanceof Error ? err.message : 'Failed to import workspace');
      setProgress(0);
    } finally {
      setIsImporting(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[80vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold text-gray-900">Import Workspace</h2>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
              disabled={isImporting}
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Import File (JSON)
            </label>
            <input
              type="file"
              accept=".json"
              onChange={handleFileChange}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              disabled={isImporting}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Workspace Name
            </label>
            <input
              type="text"
              value={workspaceName}
              onChange={(e) => setWorkspaceName(e.target.value)}
              placeholder="Enter workspace name"
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              disabled={isImporting}
            />
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {importStatus && importStatus.status === 'completed' && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg">
              <p className="text-sm text-green-700">
                Import completed successfully!
                {importStatus.imported_workspace_id && (
                  <span> Workspace ID: {importStatus.imported_workspace_id}</span>
                )}
              </p>
            </div>
          )}

          {isImporting && (
            <div className="mb-4">
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <p className="text-sm text-gray-600 mt-2">Importing... {Math.round(progress)}%</p>
            </div>
          )}

          {importHistory.length > 0 && (
            <div className="mb-4">
              <h3 className="text-sm font-medium text-gray-700 mb-2">Recent Imports</h3>
              <div className="space-y-2 max-h-40 overflow-y-auto">
                {importHistory.map((imp) => (
                  <div
                    key={imp.id}
                    className="flex justify-between items-center p-2 bg-gray-50 rounded"
                  >
                    <span className="text-sm">
                      {new Date(imp.created_at).toLocaleString()}
                    </span>
                    <span
                      className={`text-xs px-2 py-1 rounded ${
                        imp.status === 'completed'
                          ? 'bg-green-100 text-green-800'
                          : imp.status === 'failed'
                          ? 'bg-red-100 text-red-800'
                          : 'bg-yellow-100 text-yellow-800'
                      }`}
                    >
                      {imp.status}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex space-x-3 mt-6">
            <button
              onClick={handleImport}
              disabled={isImporting || !importFile}
              className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white px-4 py-2 rounded-lg font-medium transition-colors"
            >
              {isImporting ? 'Importing...' : 'Import'}
            </button>
            <button
              onClick={onClose}
              disabled={isImporting}
              className="flex-1 bg-gray-200 hover:bg-gray-300 disabled:bg-gray-100 text-gray-800 px-4 py-2 rounded-lg font-medium transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export { WorkspaceExportModal, WorkspaceImportModal };
