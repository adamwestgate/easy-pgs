import React, { useState } from 'react';

export type PGSMetadata = {
  "Polygenic Score (PGS) ID": string;
  "Publication (PMID)"?: string;
  "Release Date"?: string;
  "Reported Trait": string;
  "Number of Variants": number | string;
  "Ancestry Distribution (%) - PGS Evaluation": string;
};

export type TraitResult = {
  id: string;
  label: string;
  description: string;
  url: string;
  metadata: PGSMetadata[];
};

type ResultBlockProps = {
  result: TraitResult;
  idx: number;
  expanded: boolean;
  onHide: () => void;
  onSelect: () => void;
  onDownload?: (selectedIds: string[]) => void; // NEW!
};

const formatAncestry = (raw: string) =>
  raw
    .split('|')
    .map(seg => {
      let [region, value] = seg.split(':');
      region = region.trim();
      value = value.trim();
      return `${region}: ${value}%`;
    })
    .join('\n');

const ResultBlock: React.FC<ResultBlockProps> = ({
  result: r,
  expanded,
  onHide,
  onSelect,
  onDownload,
}) => {
  const [checkedMap, setCheckedMap] = useState<Record<string, boolean>>({});

  const toggleCheck = (id: string, yes: boolean) => {
    setCheckedMap(prev => ({ ...prev, [id]: yes }));
  };

  const anyChecked = Object.values(checkedMap).some(Boolean);

  return (
    <div
      className={`flex flex-col h-full bg-white/95 rounded-xl shadow-md p-5 mb-4 border border-gray-100 transition-all font-serif w-full
        ${expanded ? 'max-w-3xl mx-auto' : 'max-w-lg'}
      `}
    >
      {/* Main info */}
      <div>
        <h2 className="text-xl font-serif font-semibold uppercase mb-2 tracking-tight">{r.label}</h2>
        <hr className="mb-3" />
        <p className="text-gray-700 font-serif mb-4 text-[15px]">{r.description}</p>
        <a
          href={r.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-blue-500 font-serif underline mb-4 block text-sm"
        >
          View ontology trait
        </a>
      </div>
      {!expanded && (
        <button
          onClick={onSelect}
          className="font-sans mt-auto px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-base font-semibold shadow-sm transition-all"
        >
          Select
        </button>
      )}
      {/* Metadata */}
      {expanded && (
        <div className="flex flex-col gap-4 mt-6">
          {r.metadata.length > 0 ? (
            r.metadata.map((meta, jdx) => {
              const id = meta['Polygenic Score (PGS) ID'];
              const isChecked = !!checkedMap[id];
              return (
                <button
                  key={jdx}
                  type="button"
                  onClick={() => toggleCheck(id, !isChecked)}
                  className={`relative grid grid-cols-1 md:grid-cols-3 gap-4 p-4 rounded border min-h-[90px] items-center w-full transition
                    text-left focus:outline-none focus:ring-2 focus:ring-blue-400
                    ${isChecked ? "bg-blue-50 border-blue-600 ring-2 ring-blue-400" : "bg-gray-50 border-gray-200"}
                    hover:shadow-md hover:bg-blue-100/30`
                  }
                >
                  {/* 1st column: ID, trait, optional check icon */}
                  <div className="flex flex-col items-start justify-center h-full">
                    <span className="flex items-center gap-2 font-bold text-sm">
                      {id}
                      {isChecked && (
                        <svg className="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" strokeWidth={3} viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                    </span>
                    <span className="text-xs text-gray-500 mt-1">{meta['Reported Trait']}</span>
                  </div>
                  {/* 2nd column: Variants and date */}
                  <div className="text-[15px] flex flex-col gap-1">
                    <div>
                      <strong>Variants:</strong> {meta['Number of Variants']}
                    </div>
                    <div>
                      <strong>Date:</strong> {meta['Release Date'] || '—'}
                    </div>
                    <a
                      href={`https://www.pgscatalog.org/score/${id}/`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline text-xs mt-1"
                    >
                      View on PGS Catalog
                    </a>
                  </div>
                  {/* 3rd column: Ancestry */}
                  <div className="text-[14px] whitespace-pre-line break-words">
                    <strong>Ancestry:</strong>
                    <br />
                    {formatAncestry(meta['Ancestry Distribution (%) - PGS Evaluation'])}
                  </div>
                </button>
              );
            })
          ) : (
            <p className="italic text-gray-500 text-sm">No PGS metadata found for this trait.</p>
          )}
          {anyChecked && onDownload && (
            <button
              type="button"
              onClick={() => {
                const selectedIds = Object.keys(checkedMap).filter(id => checkedMap[id]);
                onDownload(selectedIds);
              }}
              className="font-sans block mx-auto mt-5 px-8 py-3 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-lg font-semibold shadow-md transition transform active:scale-95"
            >
              Next
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default ResultBlock;
