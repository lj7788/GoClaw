import { useState, useEffect } from 'react';
import {
  Brain,
  Search,
  Plus,
  Trash2,
  X,
  Filter,
} from 'lucide-react';
import type { MemoryEntry } from '@/types/api';
import { getMemory, storeMemory, deleteMemory } from '@/lib/api';

function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max) + '...';
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString();
}

export default function Memory() {
  const [entries, setEntries] = useState<MemoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [showBatchDeleteConfirm, setShowBatchDeleteConfirm] = useState(false);

  // Form state
  const [formKey, setFormKey] = useState('');
  const [formContent, setFormContent] = useState('');
  const [formCategory, setFormCategory] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const fetchEntries = (q?: string, cat?: string) => {
    setLoading(true);
    getMemory(q || undefined, cat || undefined)
      .then(setEntries)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchEntries();
  }, []);

  const handleSearch = () => {
    fetchEntries(search, categoryFilter);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch();
  };

  const categories = Array.from(new Set(entries.map((e) => e.category))).sort();

  const handleAdd = async () => {
    if (!formKey.trim() || !formContent.trim()) {
      setFormError('键和内容不能为空');
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await storeMemory(
        formKey.trim(),
        formContent.trim(),
        formCategory.trim() || undefined,
      );
      fetchEntries(search, categoryFilter);
      setShowForm(false);
      setFormKey('');
      setFormContent('');
      setFormCategory('');
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : '存储内存失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (key: string) => {
    try {
      const result = await deleteMemory(key);
      if (result.deleted) {
        setEntries((prev) => prev.filter((e) => e.key !== key));
        setSelectedKeys((prev) => {
          const newSet = new Set(prev);
          newSet.delete(key);
          return newSet;
        });
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '删除内存失败');
    } finally {
      setConfirmDelete(null);
    }
  };

  const handleBatchDelete = async () => {
    const keysToDelete = Array.from(selectedKeys);
    if (keysToDelete.length === 0) return;

    try {
      for (const key of keysToDelete) {
        const result = await deleteMemory(key);
        if (!result.deleted) {
          throw new Error(`删除内存键 ${key} 失败`);
        }
      }
      setEntries((prev) => prev.filter((e) => !selectedKeys.has(e.key)));
      setSelectedKeys(new Set());
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '批量删除失败');
    } finally {
      setShowBatchDeleteConfirm(false);
    }
  };

  const handleSelectAll = () => {
    if (selectedKeys.size === entries.length) {
      setSelectedKeys(new Set());
    } else {
      setSelectedKeys(new Set(entries.map((e) => e.key)));
    }
  };

  const handleSelectKey = (key: string) => {
    setSelectedKeys((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(key)) {
        newSet.delete(key);
      } else {
        newSet.add(key);
      }
      return newSet;
    });
  };

  if (error && entries.length === 0) {
    return (
      <div className="p-6">
        <div className="rounded-lg bg-red-900/30 border border-red-700 p-4 text-red-300">
          加载内存失败: {error}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Brain className="h-5 w-5 text-blue-400" />
          <h2 className="text-base font-semibold text-white">
            内存 ({entries.length})
          </h2>
        </div>
        <div className="flex items-center gap-3">
          {selectedKeys.size > 0 && (
            <button
              onClick={() => setShowBatchDeleteConfirm(true)}
              className="flex items-center gap-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
            >
              <Trash2 className="h-4 w-4" />
              批量删除 ({selectedKeys.size})
            </button>
          )}
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            <Plus className="h-4 w-4" />
            添加内存
          </button>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="搜索内存条目..."
            className="w-full bg-gray-900 border border-gray-700 rounded-lg pl-10 pr-4 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div className="relative">
          <Filter className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
          <select
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
            className="bg-gray-900 border border-gray-700 rounded-lg pl-10 pr-8 py-2.5 text-sm text-white appearance-none focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
          >
            <option value="">All Categories</option>
            {categories.map((cat) => (
              <option key={cat} value={cat}>
                {cat}
              </option>
            ))}
          </select>
        </div>
        <button
          onClick={handleSearch}
          className="px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          搜索
        </button>
      </div>

      {/* Error banner (non-fatal) */}
      {error && (
        <div className="rounded-lg bg-red-900/30 border border-red-700 p-3 text-sm text-red-300">
          加载内存失败: {error}
        </div>
      )}

      {/* Batch Delete Confirmation Modal */}
      {showBatchDeleteConfirm && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">批量删除确认</h3>
              <button
                onClick={() => setShowBatchDeleteConfirm(false)}
                className="text-gray-400 hover:text-white transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <p className="text-gray-300 mb-6">
              确定要删除选中的 {selectedKeys.size} 个内存条目吗？此操作无法撤销。
            </p>

            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowBatchDeleteConfirm(false)}
                className="px-4 py-2 text-sm font-medium text-gray-300 hover:text-white border border-gray-700 rounded-lg hover:bg-gray-800 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleBatchDelete}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg transition-colors"
              >
                确认删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Memory Form Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Add Memory</h3>
              <button
                onClick={() => {
                  setShowForm(false);
                  setFormError(null);
                }}
                className="text-gray-400 hover:text-white transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {formError && (
              <div className="mb-4 rounded-lg bg-red-900/30 border border-red-700 p-3 text-sm text-red-300">
                {formError}
              </div>
            )}

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Key <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  value={formKey}
                  onChange={(e) => setFormKey(e.target.value)}
                  placeholder="例如：user_preferences"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  内容 <span className="text-red-400">*</span>
                </label>
                <textarea
                  value={formContent}
                  onChange={(e) => setFormContent(e.target.value)}
                  placeholder="内存内容..."
                  rows={4}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  分类 (可选)
                </label>
                <input
                  type="text"
                  value={formCategory}
                  onChange={(e) => setFormCategory(e.target.value)}
                  placeholder="例如：preferences, context, facts"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>

            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => {
                  setShowForm(false);
                  setFormError(null);
                }}
                className="px-4 py-2 text-sm font-medium text-gray-300 hover:text-white border border-gray-700 rounded-lg hover:bg-gray-800 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleAdd}
                disabled={submitting}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50"
              >
                {submitting ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Memory Table */}
      {loading ? (
        <div className="flex items-center justify-center h-32">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
        </div>
      ) : entries.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
          <Brain className="h-10 w-10 text-gray-600 mx-auto mb-3" />
          <p className="text-gray-400">暂无内存条目。</p>
        </div>
      ) : (
        <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800">
                <th className="text-left px-4 py-3 text-gray-400 font-medium">
                  <input
                    type="checkbox"
                    checked={entries.length > 0 && selectedKeys.size === entries.length}
                    onChange={handleSelectAll}
                    className="rounded border-gray-600 text-blue-600 focus:ring-blue-500"
                  />
                </th>
                <th className="text-left px-4 py-3 text-gray-400 font-medium">
                  键
                </th>
                <th className="text-left px-4 py-3 text-gray-400 font-medium">
                  内容
                </th>
                <th className="text-left px-4 py-3 text-gray-400 font-medium">
                  分类
                </th>
                <th className="text-left px-4 py-3 text-gray-400 font-medium">
                  时间戳
                </th>
                <th className="text-right px-4 py-3 text-gray-400 font-medium">
                  操作  
                </th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr
                  key={entry.id}
                  className="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
                >
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={selectedKeys.has(entry.key)}
                      onChange={() => handleSelectKey(entry.key)}
                      className="rounded border-gray-600 text-blue-600 focus:ring-blue-500"
                    />
                  </td>
                  <td className="px-4 py-3 text-white font-medium font-mono text-xs">
                    {entry.key}
                  </td>
                  <td className="px-4 py-3 text-gray-300 max-w-[300px]">
                    <span title={entry.content}>
                      {truncate(entry.content, 80)}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-300 capitalize">
                      {entry.category}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-xs whitespace-nowrap">
                    {formatDate(entry.timestamp)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {confirmDelete === entry.key ? (
                      <div className="flex items-center justify-end gap-2">
                        <span className="text-xs text-red-400">确认删除吗？</span>
                        <button
                          onClick={() => handleDelete(entry.key)}
                          className="text-red-400 hover:text-red-300 text-xs font-medium"
                        >
                          是
                        </button>
                        <button
                          onClick={() => setConfirmDelete(null)}
                          className="text-gray-400 hover:text-white text-xs font-medium"
                        >
                          否
                        </button>
                      </div>
                    ) : (
                      <button
                        onClick={() => setConfirmDelete(entry.key)}
                        className="text-gray-400 hover:text-red-400 transition-colors"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
