// src/config.ts

/** Base URL for your Go backend during local dev */
export const API_BASE = "http://localhost:8080";

/** All of your API endpoints in one place */
export const ENDPOINTS = {
  upload:   `${API_BASE}/upload-kit`,
  search:   (q: string) => `${API_BASE}/search?q=${encodeURIComponent(q)}`,
  status:   `${API_BASE}/status`,
  download: `${API_BASE}/download`,
  results:  (kitId: string) =>
    `${API_BASE}/results?kitId=${encodeURIComponent(kitId)}`,
};
