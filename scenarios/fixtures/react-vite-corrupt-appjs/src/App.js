diff --git a/tailwind.config.js b/tailwind.config.js
index 1234567..89abcdef 100644
--- a/tailwind.config.js
+++ b/tailwind.config.js
@@ -1,4 +1,4 @@
 module.exports = {
-  content: [],
+  content: ['./src/**/*.{js,ts,jsx,tsx}'],
   theme: { extend: {} },
 };
