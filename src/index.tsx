import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import KitUpload from './KitUpload';
import SearchPage from './SearchPGS';
import LoadingPage from './LoadingPage';
import ResultsPage from './ResultsPage';
import reportWebVitals from './reportWebVitals';

document.title = 'EasyPGS'

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<KitUpload />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/loading" element={<LoadingPage />} />
        <Route path="/results" element={<ResultsPage />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>
);

reportWebVitals();
