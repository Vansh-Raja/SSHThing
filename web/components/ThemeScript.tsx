// Inlined into <head> to set the theme class BEFORE first paint.
// Avoids a flash of the wrong theme on reload.

const themeInit = `
(function () {
  try {
    var stored = localStorage.getItem("sshthing-theme");
    var prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
    var theme = stored || (prefersDark ? "dark" : "light");
    if (theme === "dark") {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  } catch (e) {}
})();
`;

export default function ThemeScript() {
  return <script dangerouslySetInnerHTML={{ __html: themeInit }} />;
}
