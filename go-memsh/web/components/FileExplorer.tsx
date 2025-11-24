'use client';

import { useState, useEffect } from 'react';
import { SessionInfo, FileNode } from '@/types/api';
import { WebSocketClient } from '@/lib/websocket-client';
import { apiClient } from '@/lib/api-client';

interface FileExplorerProps {
  session: SessionInfo;
  wsClient: WebSocketClient | null;
  onImportExport: (type: 'import' | 'export', isDir: boolean, path?: string) => void;
}

export default function FileExplorer({ session, wsClient, onImportExport }: FileExplorerProps) {
  const [fileTree, setFileTree] = useState<FileNode | null>(null);
  const [selectedPath, setSelectedPath] = useState<string>('/');
  const [selectedFiles, setSelectedFiles] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadFileTree();
  }, [session.id]);

  const loadFileTree = async () => {
    if (!wsClient) return;

    try {
      setLoading(true);
      setError(null);

      // Get directory listing using ls -laR
      const result = await wsClient.executeCommand(session.id, 'find', [
        '/',
        '-type', 'd,f',
      ]);

      if (result.error) {
        setError(result.error);
        return;
      }

      // Build tree from paths
      const tree = buildTreeFromPaths(result.output);
      setFileTree(tree);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load file tree');
    } finally {
      setLoading(false);
    }
  };

  const buildTreeFromPaths = (paths: string[]): FileNode => {
    const root: FileNode = {
      name: '/',
      path: '/',
      isDir: true,
      children: [],
      expanded: true,
    };

    const nodeMap = new Map<string, FileNode>();
    nodeMap.set('/', root);

    // Sort paths to ensure parents are processed before children
    const sortedPaths = paths.filter(p => p && p !== '/').sort();

    for (const path of sortedPaths) {
      const parts = path.split('/').filter(Boolean);
      let currentPath = '';

      for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const parentPath = currentPath || '/';
        currentPath = currentPath + '/' + part;
        const isLast = i === parts.length - 1;

        if (!nodeMap.has(currentPath)) {
          const node: FileNode = {
            name: part,
            path: currentPath,
            isDir: !isLast, // Assume directories if not the last part
            children: [],
            expanded: false,
          };

          const parent = nodeMap.get(parentPath);
          if (parent && parent.children) {
            parent.children.push(node);
          }

          nodeMap.set(currentPath, node);
        }
      }
    }

    return root;
  };

  const toggleExpand = async (node: FileNode) => {
    if (!node.isDir) return;

    // Toggle expansion
    node.expanded = !node.expanded;
    setFileTree({ ...fileTree! });

    // Load children if not loaded yet
    if (node.expanded && node.children?.length === 0) {
      await loadDirectory(node);
    }
  };

  const loadDirectory = async (node: FileNode) => {
    if (!wsClient) return;

    try {
      const result = await wsClient.executeCommand(session.id, 'ls', [
        '-la',
        node.path,
      ]);

      if (result.error) {
        console.error('Failed to load directory:', result.error);
        return;
      }

      // Parse ls output
      const children: FileNode[] = [];
      for (const line of result.output) {
        // Skip total line and . and ..
        if (line.startsWith('total') || line.endsWith(' .') || line.endsWith(' ..')) {
          continue;
        }

        // Parse ls -la output: drwxr-xr-x 2 user user 0 Jan 1 12:00 filename
        const match = line.match(/^([d-])[\w-]+\s+\d+\s+\w+\s+\w+\s+(\d+)\s+\w+\s+\d+\s+[\d:]+\s+(.+)$/);
        if (match) {
          const [, type, size, name] = match;
          const childPath = node.path === '/' ? `/${name}` : `${node.path}/${name}`;
          children.push({
            name,
            path: childPath,
            isDir: type === 'd',
            size: parseInt(size, 10),
            children: [],
            expanded: false,
          });
        }
      }

      node.children = children;
      setFileTree({ ...fileTree! });
    } catch (err) {
      console.error('Failed to load directory:', err);
    }
  };

  const handleFileSelect = (path: string, isMulti: boolean) => {
    if (isMulti) {
      setSelectedFiles((prev) =>
        prev.includes(path) ? prev.filter((p) => p !== path) : [...prev, path]
      );
    } else {
      setSelectedPath(path);
      setSelectedFiles([path]);
    }
  };

  const renderTreeNode = (node: FileNode, level: number = 0) => {
    const isSelected = selectedFiles.includes(node.path);

    return (
      <div key={node.path}>
        <div
          className={`tree-node ${isSelected ? 'selected' : ''}`}
          style={{ paddingLeft: `${level * 20}px` }}
          onClick={(e) => {
            if (node.isDir) {
              toggleExpand(node);
            }
            handleFileSelect(node.path, e.ctrlKey || e.metaKey);
          }}
        >
          {node.isDir && (
            <span className="expand-icon">
              {node.expanded ? '▼' : '▶'}
            </span>
          )}
          <span className={`file-icon ${node.isDir ? 'folder' : 'file'}`}>
            {node.isDir ? '📁' : '📄'}
          </span>
          <span className="file-name">{node.name}</span>
        </div>
        {node.isDir && node.expanded && node.children && (
          <div className="tree-children">
            {node.children.map((child) => renderTreeNode(child, level + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="file-explorer">
      <div className="file-explorer-header">
        <h3>File Explorer</h3>
        <div className="file-actions">
          <button
            onClick={() => onImportExport('import', false)}
            className="btn btn-sm"
            disabled={!wsClient}
            title="Import file"
          >
            📄↑ Import File
          </button>
          <button
            onClick={() => onImportExport('import', true)}
            className="btn btn-sm"
            disabled={!wsClient}
            title="Import directory"
          >
            📁↑ Import Dir
          </button>
          <button
            onClick={() => onImportExport('export', false, selectedPath)}
            className="btn btn-sm"
            disabled={!wsClient || selectedFiles.length === 0}
            title="Export selected file"
          >
            📄↓ Export File
          </button>
          <button
            onClick={() => onImportExport('export', true, selectedPath)}
            className="btn btn-sm"
            disabled={!wsClient || selectedFiles.length === 0}
            title="Export selected directory"
          >
            📁↓ Export Dir
          </button>
          <button
            onClick={loadFileTree}
            className="btn btn-sm"
            disabled={loading}
            title="Refresh"
          >
            🔄
          </button>
        </div>
      </div>

      {error && (
        <div className="error-message">{error}</div>
      )}

      <div className="file-explorer-body">
        {loading && !fileTree ? (
          <div className="loading">Loading files...</div>
        ) : fileTree ? (
          <div className="file-tree">
            {renderTreeNode(fileTree)}
          </div>
        ) : (
          <div className="empty-state">No files to display</div>
        )}
      </div>

      {selectedFiles.length > 0 && (
        <div className="file-info">
          <div className="info-label">Selected:</div>
          <div className="info-value">{selectedFiles.join(', ')}</div>
        </div>
      )}
    </div>
  );
}
