'use client';

import { useState, useRef } from 'react';
import { SessionInfo } from '@/types/api';
import { WebSocketClient } from '@/lib/websocket-client';

interface ImportExportDialogProps {
  session: SessionInfo;
  wsClient: WebSocketClient | null;
  type: 'import' | 'export';
  isDir: boolean;
  path?: string;
  onClose: () => void;
}

export default function ImportExportDialog({
  session,
  wsClient,
  type,
  isDir,
  path,
  onClose,
}: ImportExportDialogProps) {
  const [targetPath, setTargetPath] = useState(path || session.cwd);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleImport = async () => {
    if (!wsClient || !fileInputRef.current?.files) return;

    const files = Array.from(fileInputRef.current.files);
    if (files.length === 0) {
      setError('Please select a file or directory');
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      if (isDir) {
        // Import directory: read all files and create structure
        for (const file of files) {
          const relativePath = (file as any).webkitRelativePath || file.name;
          const targetFile = `${targetPath}/${relativePath}`;

          // Read file content
          const content = await readFileAsBase64(file);

          // Create directory structure
          const dirPath = targetFile.substring(0, targetFile.lastIndexOf('/'));
          await wsClient.executeCommand(session.id, 'mkdir', ['-p', dirPath]);

          // Import file
          await wsClient.executeCommand(session.id, 'import-file', [
            content,
            targetFile,
          ]);
        }
        setSuccess(`Imported ${files.length} files to ${targetPath}`);
      } else {
        // Import single file
        const file = files[0];
        const content = await readFileAsBase64(file);
        const targetFile = `${targetPath}/${file.name}`;

        await wsClient.executeCommand(session.id, 'import-file', [
          content,
          targetFile,
        ]);
        setSuccess(`Imported ${file.name} to ${targetFile}`);
      }

      // Close dialog after short delay
      setTimeout(() => onClose(), 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed');
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async () => {
    if (!wsClient || !path) return;

    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      if (isDir) {
        // Export directory as tar.gz
        const result = await wsClient.executeCommand(session.id, 'export-dir', [path]);

        if (result.error) {
          throw new Error(result.error);
        }

        if (result.output.length === 0) {
          throw new Error('No output from export-dir command');
        }

        // Decode base64 content
        const base64Content = result.output.join('');
        downloadBase64File(base64Content, `${path.split('/').pop()}.tar.gz`, 'application/gzip');
        setSuccess(`Exported directory ${path}`);
      } else {
        // Export single file
        const result = await wsClient.executeCommand(session.id, 'export-file', [path]);

        if (result.error) {
          throw new Error(result.error);
        }

        if (result.output.length === 0) {
          throw new Error('No output from export-file command');
        }

        // Decode base64 content
        const base64Content = result.output.join('');
        downloadBase64File(base64Content, path.split('/').pop() || 'file', 'application/octet-stream');
        setSuccess(`Exported file ${path}`);
      }

      // Close dialog after short delay
      setTimeout(() => onClose(), 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setLoading(false);
    }
  };

  const readFileAsBase64 = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        // Remove data URL prefix
        const base64 = result.split(',')[1];
        resolve(base64);
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  };

  const downloadBase64File = (base64: string, filename: string, mimeType: string) => {
    // Convert base64 to blob
    const byteCharacters = atob(base64);
    const byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    const byteArray = new Uint8Array(byteNumbers);
    const blob = new Blob([byteArray], { type: mimeType });

    // Create download link
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="dialog-overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <div className="dialog-header">
          <h3>
            {type === 'import' ? '↑' : '↓'}{' '}
            {type === 'import' ? 'Import' : 'Export'}{' '}
            {isDir ? 'Directory' : 'File'}
          </h3>
          <button onClick={onClose} className="dialog-close">
            ×
          </button>
        </div>

        <div className="dialog-body">
          {type === 'import' ? (
            <>
              <div className="form-group">
                <label>Target Path:</label>
                <input
                  type="text"
                  value={targetPath}
                  onChange={(e) => setTargetPath(e.target.value)}
                  placeholder="/path/to/destination"
                  className="form-input"
                />
              </div>

              <div className="form-group">
                <label>
                  Select {isDir ? 'Directory' : 'File'}:
                </label>
                <input
                  ref={fileInputRef}
                  type="file"
                  {...(isDir ? { webkitdirectory: '', directory: '' } : {})}
                  multiple={isDir}
                  className="form-input"
                />
              </div>
            </>
          ) : (
            <div className="export-info">
              <p>
                <strong>Path:</strong> {path}
              </p>
              <p>
                {isDir
                  ? 'The directory will be exported as a tar.gz archive.'
                  : 'The file will be downloaded to your computer.'}
              </p>
            </div>
          )}

          {error && <div className="error-message">{error}</div>}
          {success && <div className="success-message">{success}</div>}
        </div>

        <div className="dialog-footer">
          <button onClick={onClose} className="btn btn-secondary" disabled={loading}>
            Cancel
          </button>
          <button
            onClick={type === 'import' ? handleImport : handleExport}
            className="btn btn-primary"
            disabled={loading || !wsClient}
          >
            {loading ? 'Processing...' : type === 'import' ? 'Import' : 'Export'}
          </button>
        </div>
      </div>
    </div>
  );
}
