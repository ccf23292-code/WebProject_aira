'use client';

import { useState } from 'react';

interface PasswordInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  onKeyDown?: React.KeyboardEventHandler<HTMLInputElement>;
}

export function PasswordInput({
  value,
  onChange,
  placeholder,
  onKeyDown,
}: PasswordInputProps) {
  const [visible, setVisible] = useState(false);

  return (
    <div className="relative">
      <input
        type={visible ? 'text' : 'password'}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        onKeyDown={onKeyDown}
        className="w-full rounded-md border border-gray-200 px-3 py-2 pr-11 text-sm outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
      />
      <button
        type="button"
        onClick={() => setVisible((prev) => !prev)}
        className="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-gray-400 hover:text-gray-600"
        aria-label={visible ? '隐藏密码' : '显示密码'}
      >
        {visible ? <EyeOffIcon /> : <EyeIcon />}
      </button>
    </div>
  );
}

function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function EyeOffIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M3 3l18 18" />
      <path d="M10.6 10.6A3 3 0 0 0 13.4 13.4" />
      <path d="M9.9 5.2A11.7 11.7 0 0 1 12 5c6.5 0 10 7 10 7a15.5 15.5 0 0 1-4.1 4.8" />
      <path d="M6.2 6.2A15.2 15.2 0 0 0 2 12s3.5 7 10 7a10.8 10.8 0 0 0 5.7-1.6" />
    </svg>
  );
}
