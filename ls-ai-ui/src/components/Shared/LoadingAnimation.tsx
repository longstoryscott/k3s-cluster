import { useTheme } from "@mui/material";

interface IconProps {
  size?: number;
}

const Icon: React.FC<IconProps> = ({size}) => {
  const theme = useTheme();
  return (
    <svg viewBox={`0 0 ${size} ${size}`} width={size} height={size} xmlns="http://www.w3.org/2000/svg">
      <style>
        {`
    svg {
      opacity: 0;
      animation: fadeIn 2s ease-out forwards;
    }
    .trace {
      stroke: #${theme.palette.secondary.main};
      stroke-width: 1.5;
      fill: none;
      stroke-linecap: round;
      stroke-linejoin: round;
      stroke-dasharray: 1000;
      stroke-dashoffset: 1000;
      animation: draw 2s ease-in-out infinite alternate;
      filter: url(#glow);
      animation-delay: var(--delay, 0s);
    }
    .node {
      fill: #${theme.palette.secondary.dark};
      animation: glowPulse 2s ease-in-out infinite alternate;
      filter: url(#glow);
    }
    @keyframes draw {
      from {
        stroke-dashoffset: 1000;
      }
      to {
        stroke-dashoffset: 0;
      }
    }
    @keyframes fadeIn {
      to {
        opacity: 1;
      }
      from {
        opacity: 0;
      }
    }
    @keyframes glowPulse {
      from { r: 2; opacity: 0.5; }
      to { r: 3; opacity: 1; }
    }
    `}
      </style>
      <defs>
        <filter id="glow">
          <feGaussianBlur in="SourceGraphic" stdDeviation="2.5" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      <g>
        <path className="trace" d="M200 300 L190 290 L180 280 L170 270 L160 260 L150 250 L140 240 L130 230 L120 220 L110 210 L100 200 L90 190" />
        <circle className="node" cx="90" cy="190" r="2" />

        <path className="trace" d="M200 300 L210 290 L220 280 L230 270 L240 260 L250 250 L260 240 L270 230 L280 220 L290 210 L300 200 L310 190" />
        <circle className="node" cx="310" cy="190" r="2" />

        <path className="trace" d="M200 300 L190 310 L180 320 L170 330 L160 340 L150 350 L140 360 L130 370 L120 380" />
        <circle className="node" cx="120" cy="380" r="2" />

        <path className="trace" d="M200 300 L210 310 L220 320 L230 330 L240 340 L250 350 L260 360 L270 370 L280 380" />
        <circle className="node" cx="280" cy="380" r="2" />

        <path className="trace" d="M200 300 L190 290 L180 300 L170 310 L160 320 L150 330 L140 340 L130 350 L120 360 L110 370" />
        <circle className="node" cx="110" cy="370" r="2" />

        <path className="trace" d="M200 300 L210 290 L220 300 L230 310 L240 320 L250 330 L260 340 L270 350 L280 360 L290 370" />
        <circle className="node" cx="290" cy="370" r="2" />

        <path className="trace" d="M200 300 L190 290 L200 280 L190 270 L200 260 L190 250 L200 240 L190 230 L200 220" />
        <circle className="node" cx="200" cy="220" r="2" />

        <path className="trace" d="M200 300 L210 290 L200 280 L210 270 L200 260 L210 250 L200 240 L210 230 L200 220" />
        <circle className="node" cx="200" cy="220" r="2" />

        <path className="trace" d="M200 300 L190 290 L180 300 L170 310 L160 300 L150 290 L140 280 L130 270 L120 260" />
        <circle className="node" cx="120" cy="260" r="2" />

        <path className="trace" d="M200 300 L210 290 L220 300 L230 310 L240 300 L250 290 L260 280 L270 270 L280 260" />
        <circle className="node" cx="280" cy="260" r="2" />
      </g>
    </svg>

  );
};

export default Icon;