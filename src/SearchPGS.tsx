import React, { useState } from 'react';
import SearchBar from './components/SearchBar';
import ResultBlock, { TraitResult } from './components/ResultBlock';
import LoadingPage from './LoadingPage';
import { useNavigate } from 'react-router-dom';
import { ENDPOINTS } from "./config";

const itemsPerPage = 12;

const SearchPage: React.FC = () => {
  const [results, setResults] = useState<TraitResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [page, setPage] = useState(1);

  const handleSearch = async (q: string) => {
    setLoading(true);
    setError(null);
    setSelectedId(null);
    setPage(1);
    try {
      const res = await fetch(ENDPOINTS.search(q));
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const payload: { results?: TraitResult[] } = await res.json();
      setResults(payload.results ?? []);
    } catch (err: any) {
      setError(err.message || 'Unknown error');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSelect = (id: string) => {
    setSelectedId(id);
    setQuery('');
  };

  const handleHide = () => {
    setSelectedId(null);
  };

const navigate = useNavigate();

const handleDownload = (selectedIds: string[]) => {
  setLoading(true);
  navigate('/loading');   // ← this triggers your loading route!
  const kitId = localStorage.getItem('kitId');
  if (!kitId) {
    alert('No kitId in storage—please upload a kit first');
    setLoading(false);
    return;
  }
  fetch(ENDPOINTS.download, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ kitId, pgsIds: selectedIds }),
  })
    .then(res => {
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.json();
    })
    .then(data => {
      // Optionally navigate again, e.g., to /results
      // navigate('/results');
      console.log('Download result:', data);
    })
    .catch(err => console.error('Download request failed:', err))
    .finally(() => setLoading(false));
};


  const toRender = selectedId
    ? results.filter(r => r.id === selectedId)
    : results;
  const totalPages = Math.ceil(toRender.length / itemsPerPage);
  const pagedResults = toRender.slice((page - 1) * itemsPerPage, page * itemsPerPage);

  return (
    <div className="min-h-screen bg-gradient-to-br bg-whole-site">
      <div className="w-full max-w-6xl mx-auto px-2">
        {results.length === 0 && !loading && !error ? (
          // Centered SearchBar
          <div className="flex flex-col items-center justify-center min-h-screen">
            <SearchBar value={query} onChange={setQuery} onSearch={handleSearch} />
          </div>
        ) : (
          <>
            <div className="pt-12">
              <SearchBar value={query} onChange={setQuery} onSearch={handleSearch} />
            </div>
            {loading && <p className="text-gray-500 text-center">Loading…</p>}
            {error && <p className="text-red-500 text-center">Error: {error}</p>}

            {/* Expanded trait view */}
            {selectedId && toRender[0] && (
              <div className="flex justify-center mt-8">
                <ResultBlock
                  key={toRender[0].id}
                  result={toRender[0]}
                  idx={0}
                  expanded={true}
                  onHide={handleHide}
                  onSelect={() => {}}
                  onDownload={handleDownload} // NEW
                />
              </div>
            )}

            {/* Paginated grid */}
            {!selectedId && (
              <>
                <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-8 mt-8">
                  {pagedResults.map((r, idx) => (
                    <ResultBlock
                      key={r.id}
                      result={r}
                      idx={idx}
                      expanded={false}
                      onHide={() => {}}
                      onSelect={() => handleSelect(r.id)}
                    />
                  ))}
                </div>
                {totalPages > 1 && (
                  <div className="flex items-center justify-center gap-3 my-8 flex-wrap">
                    <button
                      className="px-4 py-2 rounded bg-blue-100 text-blue-700 font-semibold disabled:opacity-50"
                      onClick={() => setPage(p => Math.max(1, p - 1))}
                      disabled={page === 1}
                    >
                      Previous
                    </button>
                    {Array.from({ length: totalPages }).map((_, i) => {
                      if (totalPages > 6) {
                        if (i === 1 && page > 4) return <span key="start-ellipsis">…</span>;
                        if (i < page - 2 && i > 0) return null;
                        if (i > page + 1 && i < totalPages - 1) return null;
                        if (i === totalPages - 2 && page < totalPages - 3) return <span key="end-ellipsis">…</span>;
                      }
                      return (
                        <button
                          key={i}
                          className={`px-3 py-1 rounded font-semibold ${
                            page === i + 1
                              ? "bg-blue-600 text-white"
                              : "bg-blue-50 text-blue-700 hover:bg-blue-200"
                          }`}
                          onClick={() => setPage(i + 1)}
                        >
                          {i + 1}
                        </button>
                      );
                    })}
                    <button
                      className="px-4 py-2 rounded bg-blue-100 text-blue-700 font-semibold disabled:opacity-50"
                      onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                      disabled={page === totalPages}
                    >
                      Next
                    </button>
                  </div>
                )}
              </>
            )}

            {loading && <LoadingPage />}
          </>
        )}
      </div>
    </div>
  );
};

export default SearchPage;
