'use client';

import type { ChangeEvent } from 'react';
import { useRef, useState } from 'react';
import type { UploadedAsset } from '@aira/shared';
import { api } from '@/lib/api';

function buildMarkdownImage(url: string, alt: string) {
  const safeAlt = alt.trim() || 'image';
  return `![${safeAlt}](${url})`;
}

export default function MarkdownImageUploadButton(props: {
  label: string;
  altText?: string;
  onUploaded: (markdown: string, asset: UploadedAsset) => void;
}) {
  const { label, altText = 'image', onUploaded } = props;
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('type', 'image');
    formData.append('file', file);

    setUploading(true);
    try {
      const asset = await api.upload<UploadedAsset>('/files/upload', formData);
      onUploaded(buildMarkdownImage(asset.url, altText), asset);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Image upload failed';
      window.alert(message);
    } finally {
      setUploading(false);
      if (inputRef.current) inputRef.current.value = '';
    }
  };

  return (
    <>
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        disabled={uploading}
        className="rounded-md border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {uploading ? 'Uploading...' : label}
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        onChange={handleFileChange}
        className="hidden"
      />
    </>
  );
}
