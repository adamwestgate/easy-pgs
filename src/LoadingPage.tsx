// src/LoadingPage.tsx
import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ENDPOINTS } from "./config";

const POLL_INTERVAL   = 2_000; // ms

const LoadingPage: React.FC = () => {
  const [stage, setStage]   = useState<string>('');
  const [error, setError]   = useState<string>('');
  const navigate            = useNavigate();

  useEffect(() => {
    const iv = setInterval(async () => {
      try {
        const res = await fetch(ENDPOINTS.status, { credentials: 'include' });
        if (!res.ok) throw new Error(`Status API returned ${res.status}`);

        const data: { stage: string } = await res.json();
        setStage(data.stage);

        // When backend marks work “ready” → stop polling & show results page
        if (data.stage === 'ready') {
          clearInterval(iv);
          navigate('/results');
        }

        // Basic fail-fast handling for explicit error states
        if (data.stage === 'error' || data.stage === 'failed') {
          clearInterval(iv);
          setError(`Backend reported ${data.stage}`);
        }
      } catch (err: any) {
        clearInterval(iv);
        setError(err.message || 'Polling failed');
      }
    }, POLL_INTERVAL);

    return () => clearInterval(iv);
  }, [navigate]);

  if (error)        return <div className="p-4 text-red-600">Error&nbsp;• {error}</div>;
  if (!stage)       return <div className="p-4">Initializing…</div>;

  // Friendly copy for the stages we know about
  const messages: Record<string, string> = {
    downloading : 'Downloading PGS files…',
    normalizing : 'Normalizing score files…',
    scoring     : 'Scoring your kit…',
    ready       : 'Wrapping up…',
  };

  return (
    <div className="p-4 text-center text-lg">
      {messages[stage] ?? `Current status: ${stage}`}
    </div>
  );
};

export default LoadingPage;
