# lai - Mac Menu Bar AI Translator

A Mac menu bar application built in Go that uses local Ollama AI models to translate text from the clipboard.

## Features

- **Menu Bar Integration**: Lives in your Mac's menu bar, stays out of the way
- **Local AI Translation**: Uses Ollama running locally for privacy and speed
- **Clipboard Translation**: Translates text from your clipboard with one click
- **Configurable Models**: Choose which Ollama model to use for translation
- **Smart Translation**: Translates to English if text is in another language, or to Spanish if already in English

## Prerequisites

1. **Ollama**: Install and run Ollama locally
   ```bash
   # Install Ollama
   curl -fsSL https://ollama.ai/install.sh | sh
   
   # Pull a model (e.g., gemma3n:e4b)
   ollama pull gemma3n:e4b
   
   # Start Ollama server
   ollama serve
   ```

2. **Go**: Make sure you have Go installed (version 1.19 or later)

## Installation

1. Clone or download this repository
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Build the application:
   ```bash
   go build -o lai main.go
   ```

## Usage

1. **Start the app**:
   ```bash
   ./lai
   ```
   The app will start minimized in your menu bar.

2. **Translate text**:
   - Copy any text to your clipboard
   - Click on the lai icon in your menu bar
   - Select "Translate" from the menu, or click "Show" and then "Translate Clipboard"
   - The translation will appear in the app window

3. **Configure settings**:
   - Click "Show" from the menu bar icon
   - Click "Settings" to change the Ollama model
   - Default model is `llama3.2`

## Menu Bar Options

- **Show**: Opens the main application window
- **Translate**: Translates clipboard content and shows the result
- **Quit**: Exits the application

## Configuration

The app uses these default settings:
- **Ollama URL**: `http://localhost:11434/api/generate`
- **Default Model**: `gemma3n:e4b`

You can change the model through the Settings dialog in the app.

## Building for Distribution

To create a standalone app bundle for macOS:

```bash
# Build for macOS
go build -ldflags="-s -w" -o lai main.go

# Optional: Create an app bundle
mkdir -p lai.app/Contents/MacOS
cp lai lai.app/Contents/MacOS/
```

## Troubleshooting

1. **"Translation failed" error**: 
   - Make sure Ollama is running (`ollama serve`)
   - Check that the model is installed (`ollama list`)
   - Verify the model name in Settings

2. **"No text found in clipboard"**:
   - Copy some text to your clipboard first
   - The app reads from the system clipboard

3. **App doesn't appear in menu bar**:
   - Check if the app is running in the background
   - Try restarting the application

## Technical Details

- Built with [Fyne](https://fyne.io/) for the GUI
- Uses the Ollama HTTP API for AI translation
- Clipboard access via the `atotto/clipboard` library
- Cross-platform Go code with macOS-specific menu bar integration

## License

This project is open source. Feel free to modify and distribute.
