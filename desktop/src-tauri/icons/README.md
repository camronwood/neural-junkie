# App Icons

This directory contains the application icons for different platforms.

## Current Icons

- `32x32.png` - Windows taskbar icon
- `128x128.png` - macOS and Linux icon
- `128x128@2x.png` - macOS Retina display icon
- `icon.icns` - macOS bundle icon
- `icon.ico` - Windows executable icon (needs to be generated)

## Generating Windows .ico File

To generate the `icon.ico` file from the PNG, you can use:

### Online Tools
- https://convertio.co/png-ico/
- https://www.icoconverter.com/

### Command Line (ImageMagick)
```bash
convert app-icon.png -define icon:auto-resize=256,128,64,48,32,16 icon.ico
```

### macOS (sips)
```bash
sips -s format ico app-icon.png --out icon.ico
```

For now, Tauri will use the PNG fallback on Windows if the .ico is missing.

