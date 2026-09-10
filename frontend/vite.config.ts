import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    // Not 5173/5190 - avoids clashing with the sibling TWSEDailyTrading
    // project (which uses 5190) and other local projects.
    port: 5290,
  },
});
