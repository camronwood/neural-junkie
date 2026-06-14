# Frontend architecture — Collaboration Station (prior collab)

## Pages
- `index.html` — home hero, feature cards, footer
- `about.html` — team blurb, mission
- `contact.html` — form with name, email, message

## Color palette
- Background: `#ffffff` (white), sections `#f5f5f5` (gray)
- Text: `#111111` (black), muted `#666666` (gray)
- Accent: `#2563eb` (blue), alert/CTA `#dc2626` (red)

## Components
- Shared header nav linking all three pages
- Shared footer with copyright
- Contact form posts to `#` (no backend yet)

## CSS
- Single `style.css` with CSS variables for colors
- Mobile-first; max-width container 960px
