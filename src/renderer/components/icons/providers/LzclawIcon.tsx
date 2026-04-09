import React from 'react';

/** LZClaw provider mark — replace with official brand SVG when available. */
const LzclawIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    height="24"
    viewBox="0 0 24 24"
    width="24"
    xmlns="http://www.w3.org/2000/svg"
    style={{ flex: '0 0 auto', lineHeight: 1 }}
  >
    <title>LZClaw</title>
    <rect fill="＃DC143C" height="24" rx="5" width="24" />
    <rect fill="#fff" height="11" rx="0.5" width="2" x="5.25" y="6.5" />
    <rect fill="#fff" height="2" rx="0.5" width="5.5" x="5.25" y="15.5" />
    <polyline
      fill="none"
      points="9.75,7.25 17.75,7.25 9.75,16.75 17.75,16.75"
      stroke="#fff"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth="2.05"
    />
  </svg>
);

export default LzclawIcon;
