import React, { FormEvent } from 'react';

type SearchBarProps = {
  value: string;
  onChange: (q: string) => void;
  onSearch: (q: string) => void;
};

const SearchBar: React.FC<SearchBarProps> = ({ value, onChange, onSearch }) => {
  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onSearch(value.trim());
  };

  return (
    <div className="w-full flex max-w-3xl mx-auto flex-col items-center">
      <form
        onSubmit={handleSubmit}
        className="flex items-center justify-center gap-2 mb-8 w-full max-w-2xl"
      >
        <input
          type="text"
          placeholder="Search traits or polygenic scores"
          value={value}
          onChange={e => onChange(e.target.value)}
          className="border border-gray-300 rounded-xl p-4 w-full text-lg shadow focus:ring-2 focus:ring-blue-400 transition font-serif"
        />
        <button
          type="submit"
          className="p-4 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-lg font-semibold shadow transition"
        >
          Search
        </button>
      </form>
    </div>
  );
};

export default SearchBar;
