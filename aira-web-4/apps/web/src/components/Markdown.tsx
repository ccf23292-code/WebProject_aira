'use client';

import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';

export function MarkdownBlock({ content, className }: { content: string; className?: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkMath]}
      rehypePlugins={[rehypeKatex]}
      className={className}
    >
      {content}
    </ReactMarkdown>
  );
}

export function MarkdownInline({ content, className }: { content: string; className?: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkMath]}
      rehypePlugins={[rehypeKatex]}
      components={{
        p: ({ children }) => <span>{children}</span>,
      }}
      className={className}
    >
      {content}
    </ReactMarkdown>
  );
}
