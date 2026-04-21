"use client";

import { useEffect, useRef } from "react";

/**
 * Subtle terminal-style character rain in the background.
 * Very low opacity; respects prefers-reduced-motion (the canvas is hidden via CSS).
 * Uses currentColor of the body text so it adapts to the theme automatically.
 */
export default function TerminalBackground() {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const reduce =
      typeof window.matchMedia === "function" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduce) return;

    const chars =
      "01abcdef<>/$#*+=-_|\\{}[]()~^%".split("");
    const fontSize = 16;
    let width = 0;
    let height = 0;
    let columns = 0;
    let drops: number[] = [];
    let dpr = Math.max(1, window.devicePixelRatio || 1);

    function resize() {
      dpr = Math.max(1, window.devicePixelRatio || 1);
      width = window.innerWidth;
      height = window.innerHeight;
      if (!canvas) return;
      canvas.width = Math.floor(width * dpr);
      canvas.height = Math.floor(height * dpr);
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      if (!ctx) return;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      columns = Math.ceil(width / fontSize);
      drops = new Array(columns).fill(0).map(() => Math.random() * -50);
    }

    function readColor() {
      // Read current text color from the root so it adapts to theme.
      const style = getComputedStyle(document.documentElement);
      const ink = style.getPropertyValue("--ink").trim() || "#0a0a0a";
      const accent = style.getPropertyValue("--accent").trim() || "#1f5d3a";
      return { ink, accent };
    }

    let frame = 0;
    let raf = 0;
    let lastPaint = 0;
    const frameInterval = 1000 / 18; // ~18fps — slow, atmospheric

    function draw(ts: number) {
      if (!ctx || !canvas) return;
      if (ts - lastPaint < frameInterval) {
        raf = requestAnimationFrame(draw);
        return;
      }
      lastPaint = ts;
      frame += 1;

      const { ink, accent } = readColor();

      // Fade the previous frame slightly — creates the trailing effect.
      ctx.globalCompositeOperation = "source-over";
      ctx.fillStyle = "rgba(0,0,0,0)";
      ctx.clearRect(0, 0, width, height);

      ctx.font = `600 ${fontSize}px var(--font-jetbrains), ui-monospace, monospace`;
      ctx.textBaseline = "top";

      for (let i = 0; i < columns; i++) {
        const x = i * fontSize;
        const y = drops[i] * fontSize;

        // Head char in accent color, faint.
        const char = chars[Math.floor(Math.random() * chars.length)];
        ctx.fillStyle = hexToRgba(accent, 0.12);
        ctx.fillText(char, x, y);

        // Trail in ink, very faint.
        if (Math.random() > 0.6) {
          const trailChar = chars[Math.floor(Math.random() * chars.length)];
          ctx.fillStyle = hexToRgba(ink, 0.05);
          ctx.fillText(trailChar, x, y - fontSize);
        }

        drops[i] += 0.35 + Math.random() * 0.25;
        if (y > height && Math.random() > 0.975) {
          drops[i] = Math.random() * -20;
        }
      }

      raf = requestAnimationFrame(draw);
    }

    resize();
    window.addEventListener("resize", resize);
    raf = requestAnimationFrame(draw);

    return () => {
      window.removeEventListener("resize", resize);
      cancelAnimationFrame(raf);
    };
  }, []);

  return <canvas ref={canvasRef} className="bg-canvas" aria-hidden="true" />;
}

function hexToRgba(hex: string, alpha: number): string {
  // Accepts "#rrggbb", "rrggbb", or leaves non-hex alone.
  let h = hex.replace("#", "").trim();
  if (h.length === 3) {
    h = h
      .split("")
      .map((c) => c + c)
      .join("");
  }
  if (!/^[0-9a-fA-F]{6}$/.test(h)) {
    return hex;
  }
  const r = parseInt(h.slice(0, 2), 16);
  const g = parseInt(h.slice(2, 4), 16);
  const b = parseInt(h.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
