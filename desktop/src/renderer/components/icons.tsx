/**
 * Inline SVG icons. Stroke-based, 18px default. All take currentColor so
 * parent CSS controls the color.
 */
import type { SVGProps } from 'react';

const baseProps: SVGProps<SVGSVGElement> = {
  width: 18,
  height: 18,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.6,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
};

const Icon = (props: SVGProps<SVGSVGElement>) => (
  <svg {...baseProps} {...props} />
);

export const TerminalIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <rect x="3" y="4" width="18" height="16" rx="2" />
    <path d="M7 9l3 3-3 3" />
    <path d="M13 15h4" />
  </Icon>
);

export const UserIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <circle cx="12" cy="8" r="3.5" />
    <path d="M5 20c1.5-3.5 4-5 7-5s5.5 1.5 7 5" />
  </Icon>
);

export const GearIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1A2 2 0 1 1 4.3 17l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" />
  </Icon>
);

export const KeyIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <circle cx="8" cy="15" r="3" />
    <path d="M10.4 13L21 4" />
    <path d="M16 6l3 3" />
    <path d="M18 8l3 3" />
  </Icon>
);

export const TeamsIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <circle cx="9" cy="10" r="3" />
    <circle cx="17" cy="10" r="2.5" />
    <path d="M3 19c.8-2.6 3-4 6-4s5.2 1.4 6 4" />
    <path d="M14.5 15c.7-.4 1.5-.6 2.5-.6 2.2 0 3.7 1 4 3" />
  </Icon>
);

export const TokenIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <rect x="4" y="7" width="16" height="10" rx="5" />
    <circle cx="12" cy="12" r="2.5" />
  </Icon>
);

export const SignOutIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <path d="M9 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h4" />
    <path d="M16 17l5-5-5-5" />
    <path d="M21 12H9" />
  </Icon>
);

export const SearchIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={14} height={14}>
    <circle cx="11" cy="11" r="7" />
    <path d="M21 21l-4.3-4.3" />
  </Icon>
);

export const CheckIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={14} height={14}>
    <circle cx="12" cy="12" r="9" />
    <path d="M8 12l3 3 5-6" />
  </Icon>
);

export const ArrowUpIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={14} height={14}>
    <path d="M12 19V5" />
    <path d="M5 12l7-7 7 7" />
  </Icon>
);

export const CloudOffIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={14} height={14}>
    <path d="M3 3l18 18" />
    <path d="M9 5a6 6 0 0 1 11 4c1 .3 1.7 1 1.9 2" />
    <path d="M5 19h12a4 4 0 0 0 1.7-7.6" />
  </Icon>
);

export const StarIcon = (p: SVGProps<SVGSVGElement> & { filled?: boolean }) => {
  const { filled, ...rest } = p;
  return (
    <Icon {...rest} fill={filled ? 'currentColor' : 'none'}>
      <path d="M12 2.5l3 6.5 7 .9-5.2 4.7 1.5 7L12 18l-6.3 3.6 1.5-7L2 9.9l7-.9z" />
    </Icon>
  );
};

export const CopyIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={14} height={14}>
    <rect x="9" y="9" width="11" height="11" rx="2" />
    <path d="M5 15V5a2 2 0 0 1 2-2h10" />
  </Icon>
);

export const PlusIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <path d="M12 5v14" />
    <path d="M5 12h14" />
  </Icon>
);

export const ChevronRightIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={14} height={14}>
    <path d="M9 18l6-6-6-6" />
  </Icon>
);

export const ConnectIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <rect x="3" y="4" width="18" height="16" rx="2" />
    <path d="M7 9l3 3-3 3" />
    <path d="M13 15h4" />
  </Icon>
);

export const FolderIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
  </Icon>
);

export const MountIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <ellipse cx="12" cy="6" rx="8" ry="2.5" />
    <path d="M4 6v6c0 1.4 3.6 2.5 8 2.5s8-1.1 8-2.5V6" />
    <path d="M4 12v6c0 1.4 3.6 2.5 8 2.5s8-1.1 8-2.5v-6" />
  </Icon>
);

// ── Feedback icons (used by toast, banners, etc.) ──────────

export const CheckCircleIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <circle cx="12" cy="12" r="9" />
    <path d="M8 12l3 3 5-6" />
  </Icon>
);

export const XCircleIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <circle cx="12" cy="12" r="9" />
    <path d="M15 9l-6 6" />
    <path d="M9 9l6 6" />
  </Icon>
);

export const InfoIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 8v1" />
    <path d="M12 12v4" />
  </Icon>
);

export const AlertTriangleIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p} width={16} height={16}>
    <path d="M10.3 3.7a2 2 0 0 1 3.4 0l7 12A2 2 0 0 1 19 18.5H5a2 2 0 0 1-1.7-2.8z" />
    <path d="M12 10v3" />
    <path d="M12 16.5v.5" />
  </Icon>
);

export const BellIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
    <path d="M13.73 21a2 2 0 0 1-3.46 0" />
  </Icon>
);

export const HomeIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <path d="M3 11l9-8 9 8" />
    <path d="M5 10v10h14V10" />
  </Icon>
);

export const EditIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <path d="M12 20h9" />
    <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z" />
  </Icon>
);

export const PlayIcon = (p: SVGProps<SVGSVGElement>) => (
  <Icon {...p}>
    <polygon points="6 4 20 12 6 20 6 4" />
  </Icon>
);
