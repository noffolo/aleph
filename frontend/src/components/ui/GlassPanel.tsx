import React, { type ReactNode } from 'react';
import { ChevronDown } from 'lucide-react';

interface GlassPanelProps {
  children: ReactNode;
  /** Header content rendered in the always-visible bar */
  header?: ReactNode;
  /** Unique section key for expand/collapse state in uiSlice */
  sectionKey?: string;
  /** Whether the section content is expanded (controlled by uiSlice if sectionKey provided) */
  expanded?: boolean;
  /** Called when the expand/collapse toggle is clicked */
  onToggle?: () => void;
  /** Extra CSS classes on the outer container */
  className?: string;
  /** Icon to render in the header before the title */
  icon?: ReactNode;
  /** Whether this panel is the "advanced" tier — hidden behind opt-in */
  advanced?: boolean;
  /** Whether advanced sections are visible (controlled by parent) */
  showAdvanced?: boolean;
}

/**
 * GlassPanel — collapsible section container with glassmorphism styling.
 *
 * When `sectionKey` is provided, the panel is collapsible and wired to
 * `uiSlice.expandedSections`. Otherwise it renders as a simple glass container.
 */
export const GlassPanel: React.FC<GlassPanelProps> = ({
  children,
  header,
  sectionKey,
  expanded,
  onToggle,
  className = '',
  icon,
  advanced = false,
  showAdvanced = false,
}) => {
  const isCollapsible = !!sectionKey;
  const isOpen = isCollapsible ? !!expanded : true;

  if (advanced && !showAdvanced) return null;

  return (
    <div
      className={`bg-surface rounded-lg border border-border shadow-sm overflow-hidden ${className}`}
    >
      {isCollapsible && header && (
        <button
          onClick={onToggle}
          className="w-full flex items-center justify-between px-6 py-4 text-left hover:bg-surface-alt/40 transition-colors focus:outline-none focus:ring-1 focus:ring-inset focus:ring-primary/30"
          aria-expanded={isOpen}
        >
          <div className="flex items-center gap-3">
            {icon && <span className="text-primary shrink-0">{icon}</span>}
            <span className="font-semibold text-sm">{header}</span>
          </div>
          <ChevronDown
            size={16}
            className={`shrink-0 text-textMuted transition-transform duration-200 ${isOpen ? 'rotate-0' : '-rotate-90'}`}
          />
        </button>
      )}
      {isCollapsible ? (
        <div
          className={`grid transition-all duration-200 ease-in-out ${
            isOpen ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'
          }`}
        >
          <div className="overflow-hidden">
            <div className="px-6 pb-5 pt-2">{children}</div>
          </div>
        </div>
      ) : (
        children
      )}
    </div>
  );
};
