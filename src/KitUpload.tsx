import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Transition } from '@headlessui/react'
import { ENDPOINTS } from "./config";

export default function KitUpload() {
  const navigate = useNavigate()
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setError(null)
    setSuccess(null)
    const selected = e.target.files?.[0] ?? null

    if (selected && !selected.name.match(/\.(txt|gz)$/i)) {
      setError('Please select a .txt or .gz kit file.')
      setFile(null)
    } else {
      setFile(selected)
    }
  }

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!file) return

    setUploading(true)
    setError(null)
    setSuccess(null)

    const formData = new FormData()
    formData.append('kit', file)

    try {
      const res = await fetch(ENDPOINTS.upload, {
        method: 'POST',
        credentials: 'include',
        body: formData,
      })

      if (!res.ok) {
        const msg = await res.text()
        throw new Error(msg || 'Upload failed')
      }

      const data = await res.json()
      localStorage.setItem('kitId', data.kit_id)
      setSuccess('File uploaded successfully!')
      navigate('/search')
    } catch (err: any) {
      setError(err.message || 'Upload error')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-whole-site">
      <Transition
        appear
        show={true}
        enter="transition-opacity duration-700"
        enterFrom="opacity-0"
        enterTo="opacity-100"
      >
        <div className="w-full max-w-md rounded-2xl shadow-xl bg-white/80 p-16 border border-gray-100">
          <h1 className="text-2xl font-medium font-serif mb-6 text-center tracking-tight text-gray-700 ">Upload DNA Kit</h1>
          <form onSubmit={handleUpload} className="flex flex-col gap-4">
            <label
              className={`flex flex-col items-center border-2 border-dashed border-blue-300 bg-blue-50 hover:bg-blue-100 rounded-xl p-8 cursor-pointer transition ${
                !file ? "animate-breathe" : ""
              }`}
            >
              <span className="text-blue-600 font-medium font-serif mb-1">
                Choose .txt or .gz file
              </span>
              <input
                type="file"
                accept=".txt,.gz"
                onChange={handleFileChange}
                disabled={uploading}
                className="hidden"
              />
              <span className="text-xs text-gray-500 mt-2">{file ? file.name : 'No file selected.'}</span>
            </label>

            {error && <div className="text-red-500 text-center">{error}</div>}
            {success && <div className="text-green-600 text-center">{success}</div>}

            <button
              type="submit"
              disabled={!file || uploading}
              className="mt-2 py-3 rounded-xl bg-blue-500 hover:bg-blue-600 text-white text-lg font-semibold transition disabled:opacity-60"
            >
              {uploading ? 'Uploading...' : 'Upload Kit'}
            </button>
          </form>
        </div>
      </Transition>
      <div className="fixed bottom-6 right-8 opacity-30 text-xl font-serif font-semibold pointer-events-none select-none z-50">
      EasyPGS
    </div>
    </div>
  )
}
