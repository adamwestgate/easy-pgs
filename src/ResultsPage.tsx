import React, { useEffect, useState } from "react";
import { Disclosure, Transition } from "@headlessui/react";
import { ENDPOINTS } from "./config";

type ScoreMap = Record<string, number>;
interface ApiResponse {
  population: ScoreMap;
  user: ScoreMap;
  trait: Record<string, string>;
  sd?: ScoreMap;
  z?: ScoreMap;
  pct?: ScoreMap;
  pct_snps_scored?: ScoreMap;
}

function fmtScore(v: number): string {
  if (isNaN(v)) return "—";
  return v.toExponential(3);
}
function fmtZ(z: number | undefined): string {
  if (z === undefined || isNaN(z)) return "—";
  return z.toFixed(2);
}
function fmtPct(p: number | undefined): string {
  if (p === undefined || isNaN(p)) return "—";
  return (p * 100).toFixed(1) + "%";
}
function fmtCov(c: number | undefined): string {
  if (c === undefined || isNaN(c)) return "—";
  return c.toFixed(1) + "%";
}

export default function ResultsPage() {
  const [data, setData] = useState<ApiResponse | null>(null);
  const [error, setError] = useState("");
  const [show, setShow] = useState(false);
  const kitId = localStorage.getItem("kitId") || "";

  useEffect(() => {
    if (!kitId) {
      setError("No kitId found—please upload a kit first.");
      return;
    }
    (async () => {
      try {
          const res = await fetch(
            ENDPOINTS.results(kitId),
          );
        if (!res.ok) throw new Error(`Results API returned ${res.status}`);
        setData((await res.json()) as ApiResponse);
        setTimeout(() => setShow(true), 50); // Fade in after short delay
      } catch (err: any) {
        setError(err.message || "Could not load results");
      }
    })();
  }, [kitId]);

  if (error)
    return (
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-100 via-white to-purple-100">
        <div className="text-red-600 p-8 rounded-lg bg-white shadow-xl border border-gray-100">
          {error}
        </div>
      </div>
    );
  if (!data)
    return (
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-100 via-white to-purple-100">
        <div className="p-8 rounded-lg bg-white shadow-xl border border-gray-100">
          Loading results…
        </div>
      </div>
    );

  const rows = Object.keys(data.population).map((rawId) => {
    const cleanId = rawId.split(".")[0];
    return {
      rawId,
      id: cleanId,
      trait: data.trait?.[rawId] ?? data.trait?.[cleanId] ?? "—",
      user: data.user[rawId] ?? NaN,
      pop: data.population[rawId] ?? NaN,
      z: data.z?.[rawId],
      pct: data.pct?.[rawId],
      coverage: data.pct_snps_scored?.[rawId],
    };
  });

  return (
      <div className="min-h-screen flex items-center justify-center px-4 bg-whole-site">
      <Transition
        show={show}
        appear
        enter="transition-opacity duration-700"
        enterFrom="opacity-0"
        enterTo="opacity-100"
      >
        <div className="w-full max-w-4xl rounded-2xl bg-white/80 border border-gray-100 p-8 shadow-modal">
          <h1 className="text-2xl font-serif font-medium mb-8 text-center text-gray-700 tracking-tight">
            Polygenic-Score Results
          </h1>
          <div className="overflow-x-auto">
            <table className="w-full rounded-xl overflow-hidden">
              <thead>
                <tr>
                  <th className="bg-gray-50 font-serif px-5 py-3 text-left rounded-tl-xl">
                    PGS ID
                  </th>
                  <th className="bg-gray-50 font-serif px-5 py-3 text-left">
                    Reported Trait
                  </th>
                  <th className="bg-gray-50 font-serif px-5 py-3 text-center">
                    Percentile
                  </th>
                  <th className="bg-gray-50 font-serif px-5 py-3 text-center">
                    Coverage
                  </th>
                  <th className="bg-gray-50 px-4 py-2 rounded-tr-xl"></th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r, idx) => (
                  <Disclosure key={r.id}>
                    {({ open }) => (
                      <>
                        <tr
                          className={
                            "transition-all group " +
                            (open
                              ? "bg-[#e6f1fb]/80" // Soft mint highlight when expanded
                              : idx % 2 === 0
                              ? "bg-white"
                              : "bg-gray-50") +
                            " hover:bg-[#e6f1fb]/60" // Softer mint on hover
                          }
                        >
                          <td className="px-4 py-3 font-mono font-medium text-gray-700">
                            {r.id}
                          </td>
                          <td className="px-4 py-3 text-gray-700">
                            {r.trait}
                          </td>
                          <td className="px-4 py-3 text-center font-semibold text-blue-600">
                            {fmtPct(r.pct)}
                          </td>
                          <td
                            className={
                              "px-4 py-3 text-center font-semibold " +
                              ((r.coverage ?? 100) < 25
                                ? "text-red-600"
                                : "text-gray-700")
                            }
                          >
                            {fmtCov(r.coverage)}
                          </td>
                          <td className="px-4 py-3 text-center">
                            <Disclosure.Button className="rounded-lg bg-gray-200 hover:bg-blue-100 text-sm font-semibold px-4 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-all">
                              {open ? "Hide" : "Details"}
                            </Disclosure.Button>
                          </td>
                        </tr>
                        <Transition
                          show={open}
                          appear
                          enter="transition-opacity duration-500"
                          enterFrom="opacity-0"
                          enterTo="opacity-100"
                          leave="transition-opacity duration-300"
                          leaveFrom="opacity-100"
                          leaveTo="opacity-0"
                        >
                          <Disclosure.Panel as="tr">
                            <td colSpan={5} className="p-0 border-t-0">
                              <div className="flex flex-col md:flex-row items-center justify-center gap-6 p-6 bg-gray-50 border border-gray-100 rounded-b-xl">
                                <div>
                                  <div className="text-xs text-gray-500 font-mono mb-1">
                                    Your Score
                                  </div>
                                  <div className="font-semibold text-gray-700">
                                    {fmtScore(r.user)}
                                  </div>
                                </div>
                                <div>
                                  <div className="text-xs text-gray-500 font-mono mb-1">
                                    Population Mean
                                  </div>
                                  <div className="font-semibold text-gray-700">
                                    {fmtScore(r.pop)}
                                  </div>
                                </div>
                                <div>
                                  <div className="text-xs text-gray-500 font-mono mb-1">
                                    z-score
                                  </div>
                                  <div className="font-semibold text-gray-700">
                                    {fmtZ(r.z)}
                                  </div>
                                </div>
                              </div>
                            </td>
                          </Disclosure.Panel>
                        </Transition>
                      </>
                    )}
                  </Disclosure>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </Transition>
    </div>
  );
}
